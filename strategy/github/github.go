package github

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"sort"

	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/gofishgithub"
	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/printer"

	"strings"

	"github.com/google/go-github/v26/github"
	ghApi "github.com/google/go-github/v26/github"
)

type Github struct {
	GoFish *gofishgithub.GoFish
}

func (g *Github) UpdateApplications(ctx context.Context, appsGithub []models.DesiredApp, createPullrequests bool) {
	applications := []*models.Application{}

	for _, app := range appsGithub {
		application, err := g.CreateApplication(ctx, app)
		if err != nil {
			log.G(ctx).Warnf("Error in handling %s: %v", app.Repo, err)
		} else {
			currentVersion, err := g.GoFish.GetCurrentVersion(ctx, app)
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
				g.CreateLuaFile(ctx, app)

				err := g.GoFish.Lint(app)
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
				g.CreateLuaFile(ctx, app)
				g.CreatePullRequest(ctx, app)
			} else if missing {
				log.G(ctx).Infof("Will not create new apps for now: %s", app.Name)
			}
		}
	}
}

func (g *Github) CreateApplication(ctx context.Context, app models.DesiredApp) (*models.Application, error) {
	log.G(ctx).Infof("## Creating Application for %s", app.Repo)

	releaseList, _, err := g.GoFish.Client.Repositories.ListReleases(ctx, app.Org, app.Repo, &ghApi.ListOptions{})
	if err != nil {
		return nil, err
	}

	repoDetails, _, err := g.GoFish.Client.Repositories.Get(ctx, app.Org, app.Repo)
	if err != nil {
		return nil, err
	}

	release := findRelease(app, releaseList)

	releaseName := release.GetTagName()

	homepage := repoDetails.GetHomepage()
	if homepage == "" {
		homepage = repoDetails.GetHTMLURL()
	}

	var application = models.Application{
		ReleaseName:  releaseName,
		Name:         app.Repo,
		Description:  repoDetails.GetDescription(),
		Organization: app.Org,
		Path:         app.Path,
		Version:      strings.Replace(releaseName, "v", "", 1),
		Arch:         app.Arch,
		Licence:      repoDetails.GetLicense().GetSPDXID(),
		Homepage:     homepage,
		Assets:       []models.Asset{},
	}

	checksumService := NewChecksumService(application, g.GoFish.Client, release.Assets)
	application.Assets = g.GetAssets(application, release.Assets, checksumService)
	return &application, nil
}

func (g *Github) CreateLuaFile(ctx context.Context, application *models.Application) {
	f, err := os.Create("/usr/local/Fish/Rigs/github.com/fishworks/fish-food/Food/" + application.Name + ".lua")
	if err != nil {
		log.G(ctx).Warn(err)
		return
	}
	defer f.Close()
	err = serializeLuaContent(application, f)
	if err != nil {
		log.G(ctx).Warn(err)
		return
	}
}

func (g *Github) CreatePullRequest(ctx context.Context, application *models.Application) {

	log.G(ctx).Infof("## Creating Pullrequest for %s version %s", application.Name, application.Version)

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	err := serializeLuaContent(application, foo)
	if err != nil {
		log.G(ctx).Warn(err)
		return
	}
	foo.Flush()

	err = g.GoFish.CreatePullRequest(ctx, application, b.Bytes())
	if err != nil {
		log.G(ctx).Warnf("Failed creating PR: %v", err)
		return
	}
}

func findRelease(app models.DesiredApp, releaseList []*ghApi.RepositoryRelease) *ghApi.RepositoryRelease {
	if app.Onlyprerelease {
		for k, v := range releaseList {
			if v.GetPrerelease() {
				continue
			}
			return releaseList[k]
		}
	}
	return releaseList[0]

	// for _, release := range releaseList {
	// 	fmt.Printf("Release %s\n", release.GetTagName())
	// 	if !strings.Contains(release.GetTagName(), "beta") {
	// 		return release
	// 	}
	// }
	// return releaseList[0]
}

func (g *Github) GetAssets(app models.Application, releaseAssets []github.ReleaseAsset, checksumService *ChecksumService) []models.Asset {

	assets := []models.Asset{}

	for _, c := range releaseAssets {
		if strings.Contains(c.GetName(), "sha256") || strings.Contains(c.GetName(), "sha512") {
			continue
		}
		cleanName := strings.ToLower(c.GetName())
		assetName := strings.Replace(c.GetName(), app.Name, "\" .. name .. \"", 1)
		assetName = strings.Replace(assetName, app.Version, "\" .. version .. \"", 1)
		path := "name"
		if !strings.Contains(cleanName, "tar") && !strings.Contains(cleanName, "zip") {
			path = strings.Replace(c.GetName(), app.Name, "name .. \"", 1) + "\""
		}

		if (strings.Contains(cleanName, "osx") || strings.Contains(cleanName, "darwin") || strings.Contains(cleanName, "macos")) && strings.Contains(cleanName, app.Arch) {
			assets = append(assets, models.Asset{
				Arch:        "amd64",
				Os:          "darwin",
				AssertName:  assetName,
				InstallPath: "\"bin/\" .. name",
				Path:        path,
				Sha256:      checksumService.getChecksum(c.GetBrowserDownloadURL(), c.GetName()),
				Executable:  true,
			})

		} else if strings.Contains(cleanName, "linux") && strings.Contains(cleanName, app.Arch) {
			assets = append(assets, models.Asset{
				Arch:        "amd64",
				Os:          "linux",
				AssertName:  assetName,
				InstallPath: "\"bin/\" .. name",
				Path:        path,
				Sha256:      checksumService.getChecksum(c.GetBrowserDownloadURL(), c.GetName()),
				Executable:  true,
			})
		} else if (strings.Contains(cleanName, "win") || strings.Contains(cleanName, "windows")) && strings.Contains(cleanName, app.Arch) {
			if strings.Contains(cleanName, "tar") || strings.Contains(cleanName, "zip") {
				path = "name .. \".exe\""
			}
			if !strings.Contains(cleanName, "tar") && !strings.Contains(cleanName, "zip") && !strings.Contains(cleanName, "exe") {
				path = "name"
			}

			assets = append(assets, models.Asset{
				Arch:        "amd64",
				Os:          "windows",
				AssertName:  assetName,
				InstallPath: "\"bin\\\\\" .. name .. \".exe\"",
				Path:        path,
				Sha256:      checksumService.getChecksum(c.GetBrowserDownloadURL(), c.GetName()),
				Executable:  false,
			})
		}
	}

	return g.sortAssets(assets)
}

func (g *Github) sortAssets(assets []models.Asset) []models.Asset {
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Os < assets[j].Os
	})
	return assets
}
