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

	"github.com/gofish-bot/gofish-bot/printer"

	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/gofishgithub"

	"github.com/gofish-bot/gofish-bot/models"
	"github.com/sirupsen/logrus"

	"strings"

	ghApi "github.com/google/go-github/v26/github"
)

type HashiCorp struct {
	Log         *logrus.Logger
	Client      *ghApi.Client
	GithubToken string
	GoFish      *gofishgithub.GoFish
}

func (h *HashiCorp) UpdateApplications(ctx context.Context, appsGithub []models.DesiredApp, createPullrequests bool) {
	applications := []*models.Application{}

	for _, app := range appsGithub {
		application, err := h.CreateApplication(ctx, app)
		if err != nil {
			h.Log.Warnf("Error in handling %s: %v", app.Repo, err)
		} else {
			currentVersion, err := h.GoFish.GetCurrentVersion(ctx, app)
			if err != nil {
				h.Log.Warn(errors.Wrap(err, "Could not find current version"))
			} else {
				application.CurrentVersion = currentVersion
			}

			applications = append(applications, application)
		}
	}

	printer.Table(applications)

	for _, app := range applications {
		// var buf bytes.Buffer

		// err := newCreateHashicorp(app, &buf)
		// if err != nil {
		// 	h.Log.Error(err)
		// 	continue
		// }

		// err = h.GoFish.LintString(app.Name, buf.String())
		// if err != nil {
		// 	for _, line := range strings.Split(err.Error(), "\n") {
		// 		h.Log.Warnf("%v", line)
		// 	}
		// 	continue
		// }

		if app.CurrentVersion != app.Version {
			missing := app.CurrentVersion == ""
			needsUpgrade := !missing && app.CurrentVersion != app.Version
			if needsUpgrade && createPullrequests {
				h.CreatePullRequest(ctx, app)
			} else if missing {
				h.Log.Infof("Will not create new apps for now: %s", app.Name)
			}
		}
	}
}

func (h *HashiCorp) CreateApplication(ctx context.Context, app models.DesiredApp) (*models.Application, error) {
	h.Log.Infof("## Creating Application for %s", app.Repo)

	tagList, _, err := h.Client.Repositories.ListTags(ctx, app.Org, app.Repo, &ghApi.ListOptions{})
	if err != nil {
		return nil, err
	}

	repoDetails, _, err := h.Client.Repositories.Get(ctx, app.Org, app.Repo)
	if err != nil {
		return nil, err
	}

	release := tagList[0]
	var checksums string
	releaseName := release.GetName()
	cleanVersion := strings.Replace(releaseName, "v", "", 1)

	checksums, err = h.downloadFile(fmt.Sprintf("https://releases.hashicorp.com/%s/%s/%s_%s_SHA256SUMS", app.Repo, cleanVersion, app.Repo, cleanVersion))
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

func (h *HashiCorp) CreateLuaFile(application *models.Application) {
	f, err := os.Create("/usr/local/Fish/Rigs/github.com/fishworks/fish-food/Food/" + application.Name + ".lua")
	if err != nil {
		h.Log.Warn(err)
		return
	}
	defer f.Close()
	err = newCreateHashicorp(application, f)
	if err != nil {
		h.Log.Warn(err)
		return
	}
}

func (h *HashiCorp) CreatePullRequest(ctx context.Context, application *models.Application) {
	h.Log.Infof("## Creating Pullrequest for %s version %s", application.Name, application.Version)

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	newCreateHashicorp(application, foo)
	foo.Flush()

	err := h.GoFish.CreatePullRequest(ctx, application, b.Bytes())
	if err != nil {
		h.Log.Warnf("Failed creating PR: %v", err)
		return
	}
}

func (h *HashiCorp) createApplication(app models.DesiredApp, description, licence, version, currentVersion, homepage string, checksums string) models.Application {
	cleanVersion := strings.Replace(version, "v", "", 1)

	assets := []models.Asset{}
	var application = models.Application{
		ReleaseName:    version,
		Name:           app.Repo,
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
func (h *HashiCorp) downloadFile(url string) (string, error) {

	h.Log.Debugf("Downloading: %s", url)
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
