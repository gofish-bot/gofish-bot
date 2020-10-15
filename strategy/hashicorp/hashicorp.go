package hashicorp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"github.com/blang/semver"
	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/printer"

	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/gofishgithub"

	"github.com/gofish-bot/gofish-bot/models"

	"strings"

	ghApi "github.com/google/go-github/v32/github"
)

type HashiCorp struct {
	GoFish *gofishgithub.GoFish
}

func (h *HashiCorp) UpdateApplications(ctx context.Context, appsGithub []models.DesiredApp, createPullrequests bool) {
	applications := []*models.Application{}

	for _, app := range appsGithub {
		application, err := h.CreateApplication(ctx, app)
		if err != nil {
			log.G(ctx).Warnf("Error in handling %s: %v", app.Name, err)
		} else {
			currentVersion, err := h.GoFish.GetCurrentVersion(ctx, app)
			if err != nil {
				log.G(ctx).Warn(errors.Wrap(err, "Could not find current version"))
			} else {
				application.CurrentVersion = currentVersion
			}

			applications = append(applications, application)
		}
	}

	printer.Table(applications)

	for _, app := range applications {
		if app.CurrentVersion != app.Version {
			missing := app.CurrentVersion == ""
			needsUpgrade := !missing && app.CurrentVersion != app.Version
			upgradeToBeta := (!strings.Contains(app.CurrentVersion, "beta")) && strings.Contains(app.Version, "beta")

			if needsUpgrade {
				h.CreateLuaFile(ctx, app)

				err := h.GoFish.Lint(app)
				if err != nil {
					log.G(ctx).Warnf("Linting failed: %v", err)
					continue
				} else {
					log.G(ctx).Infof("Linting ok: %v", app.Name)
				}
			}

			if upgradeToBeta {
				log.G(ctx).Infof("Will not upgrade to beta release: %s", app.Name)
			} else if needsUpgrade && createPullrequests {
				log.G(ctx).Infof("Creating pr for release: %s", app.Name)
				h.CreatePullRequest(ctx, app)
			} else if missing {
				log.G(ctx).Infof("Will not create new apps for now: %s", app.Name)
			}
		}
	}
}

func (h *HashiCorp) CreateApplication(ctx context.Context, app models.DesiredApp) (*models.Application, error) {
	log.G(ctx).Infof("## Creating Application for %s: https://github.com/%s/%s/releases/", app.Name, app.Org, app.Repo)

	tagList, _, err := h.GoFish.Client.Repositories.ListTags(ctx, app.Org, app.Name, &ghApi.ListOptions{})
	if err != nil {
		return nil, err
	}

	repoDetails, _, err := h.GoFish.Client.Repositories.Get(ctx, app.Org, app.Name)
	if err != nil {
		return nil, err
	}

	release := tagList[0]

	newestRelease, _ := semver.Make("0.0.0")

	for _, v := range tagList {
		releaseVersion, err := semver.Make(strings.Replace(v.GetName(), "v", "", 1))
		if err != nil {
			continue
		}
		if len(releaseVersion.Pre) != 0 {
			log.L.Debugf("Discarding prerelease: %s", v.GetName())
		}

		if releaseVersion.GT(newestRelease) && len(releaseVersion.Pre) == 0 {
			newestRelease = releaseVersion
			release = v
		}
	}

	var checksums string
	releaseName := release.GetName()
	cleanVersion := strings.Replace(releaseName, "v", "", 1)

	checksums, err = h.downloadFile(ctx, fmt.Sprintf("https://releases.hashicorp.com/%s/%s/%s_%s_SHA256SUMS", app.Name, cleanVersion, app.Name, cleanVersion))
	if err != nil {
		return nil, errors.Wrap(err, "Could not download checksums")
	}

	homepage := repoDetails.GetHomepage()
	if homepage == "" {
		homepage = repoDetails.GetHTMLURL()
	}
	currentVersion, err := h.GoFish.GetCurrentVersion(ctx, app)
	if err != nil {
		return nil, errors.Wrap(err, "Could not find current version")
	}

	application := h.createApplication(app, repoDetails.GetDescription(), repoDetails.GetLicense().GetSPDXID(), releaseName, currentVersion, homepage, checksums)
	return &application, nil
}

func (h *HashiCorp) CreateLuaFile(ctx context.Context, application *models.Application) {
	f, err := os.Create("/usr/local/gofish/tmp/github.com/fmotrifork/fish-food/Food/" + application.Name + ".lua")
	if err != nil {
		log.G(ctx).Warn(err)
		return
	}
	defer f.Close()
	err = newCreateHashicorp(application, f)
	if err != nil {
		log.G(ctx).Warn(err)
		return
	}
}

func (h *HashiCorp) CreatePullRequest(ctx context.Context, application *models.Application) {
	log.G(ctx).Infof("## Creating Pullrequest for %s version %s", application.Name, application.Version)

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	err := newCreateHashicorp(application, foo)
	if err != nil {
		log.G(ctx).Warnf("Failed creating Lua file: %v", err)
		return
	}
	foo.Flush()

	err = h.GoFish.CreatePullRequest(ctx, application, b.Bytes())
	if err != nil {
		log.G(ctx).Warnf("Failed creating PR: %v", err)
		return
	}
}

func (h *HashiCorp) createApplication(app models.DesiredApp, description, licence, version, currentVersion, homepage string, checksums string) models.Application {
	cleanVersion := strings.Replace(version, "v", "", 1)

	assets := []models.Asset{}
	var application = models.Application{
		ReleaseName:    version,
		Name:           app.Name,
		Repo:           app.Repo,
		Description:    description,
		Organization:   app.Org,
		Version:        cleanVersion,
		Licence:        licence,
		Homepage:       homepage,
		Assets:         []models.Asset{},
		CurrentVersion: currentVersion,
	}

	assets = append(assets, models.Asset{
		Arch:        "amd64",
		Os:          "darwin",
		AssertName:  "_darwin_amd64.zip",
		InstallPath: "\"bin/\" .. name",
		Path:        "name",
		Sha256:      h.getChecksum("_darwin_amd64.zip", checksums),
		Executable:  true,
	})

	assets = append(assets, models.Asset{
		Arch:        "amd64",
		Os:          "linux",
		AssertName:  "_linux_amd64.zip",
		InstallPath: "\"bin/\" .. name",
		Path:        "name",
		Sha256:      h.getChecksum("_linux_amd64.zip", checksums),
		Executable:  true,
	})

	assets = append(assets, models.Asset{
		Arch:        "amd64",
		Os:          "windows",
		AssertName:  "_windows_amd64.zip",
		InstallPath: "\"bin\\\\\" .. name .. \".exe\"",
		Path:        "name .. \".exe\"",
		Sha256:      h.getChecksum("_windows_amd64.zip", checksums),
		Executable:  false,
	})

	application.Assets = h.sortAssets(assets)
	return application
}

func (h *HashiCorp) sortAssets(assets []models.Asset) []models.Asset {
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Os < assets[j].Os
	})
	return assets
}

func (h *HashiCorp) getChecksum(assetName, checksums string) string {
	for _, line := range strings.Split(strings.TrimSuffix(checksums, "\n"), "\n") {
		if strings.Contains(line, assetName) {
			return strings.Split(line, " ")[0]
		}
	}
	return ""
}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func (h *HashiCorp) downloadFile(ctx context.Context, url string) (string, error) {

	log.G(ctx).Debugf("Downloading: %s", url)
	// Get the data
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "nil", err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "nil", err
	}

	return string(res), nil
}
