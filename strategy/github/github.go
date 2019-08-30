package github

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"sort"

	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/gofishgithub"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/printer"
	"github.com/sirupsen/logrus"

	"strings"

	"github.com/google/go-github/v26/github"
	ghApi "github.com/google/go-github/v26/github"
)

type Github struct {
	Log         *logrus.Logger
	Client      *ghApi.Client
	GithubToken string
	GoFish      *gofishgithub.GoFish
}

func (g *Github) UpdateApplications(ctx context.Context, appsGithub []models.DesiredApp, createPullrequests bool) {
	applications := []*models.Application{}

	for _, app := range appsGithub {
		application, err := g.CreateApplication(ctx, app)
		if err != nil {
			g.Log.Warnf("Error in handling %s: %v", app.Repo, err)
		} else {
			currentVersion, err := g.GoFish.GetCurrentVersion(ctx, app)
			if err != nil {
				g.Log.Warn(errors.Wrap(err, "Could not find current version"))
			} else {
				application.CurrentVersion = currentVersion
			}

			applications = append(applications, application)
		}
	}

	printer.Table(applications)

	for _, app := range applications {
		// var buf bytes.Buffer

		// err := serializeLuaContent(app, &buf)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }

		// err = g.GoFish.LintString(app.Name, buf.String())
		// if err != nil {
		// 	g.Log.Warnf("Linting failed: %v", err)
		// 	continue
		// }
		if app.CurrentVersion != app.Version {
			missing := app.CurrentVersion == ""
			needsUpgrade := !missing && app.CurrentVersion != app.Version
			if needsUpgrade && createPullrequests {
				g.CreatePullRequest(ctx, app)
			} else if missing {
				g.Log.Infof("Will not create new apps for now: %s", app.Name)
			}
		}
	}
}

func (g *Github) CreateApplication(ctx context.Context, app models.DesiredApp) (*models.Application, error) {
	g.Log.Infof("## Creating Application for %s", app.Repo)

	releaseList, _, err := g.Client.Repositories.ListReleases(ctx, app.Org, app.Repo, &ghApi.ListOptions{})
	if err != nil {
		return nil, err
	}

	repoDetails, _, err := g.Client.Repositories.Get(ctx, app.Org, app.Repo)
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

	checksumService := NewChecksumService(application, g.GithubToken, release.Assets, g.Log)
	application.Assets = g.GetAssets(application, release.Assets, checksumService)
	return &application, nil
}

func (g *Github) CreateLuaFile(application *models.Application) {
	f, err := os.Create("/usr/local/Fish/Rigs/github.com/fishworks/fish-food/Food/" + application.Name + ".lua")
	if err != nil {
		g.Log.Warn(err)
		return
	}
	defer f.Close()
	err = serializeLuaContent(application, f)
	if err != nil {
		g.Log.Warn(err)
		return
	}
}

func (g *Github) CreatePullRequest(ctx context.Context, application *models.Application) {

	g.Log.Infof("## Creating Pullrequest for %s version %s", application.Name, application.Version)

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	err := serializeLuaContent(application, foo)
	if err != nil {
		g.Log.Warn(err)
		return
	}
	foo.Flush()

	err = g.GoFish.CreatePullRequest(ctx, application, b.Bytes())
	if err != nil {
		g.Log.Warnf("Failed creating PR: %v", err)
		return
	}
}

func findRelease(app models.DesiredApp, releaseList []*ghApi.RepositoryRelease) *ghApi.RepositoryRelease {
	release := releaseList[0]
	if app.Onlyprerelease {
		for k, v := range releaseList {
			if v.GetPrerelease() {
				continue
			}
			return releaseList[k]
		}
	}
	return release
}

func (g *Github) GetAssets(app models.Application, releaseAssets []github.ReleaseAsset, checksumService *ChecksumService) []models.Asset {

	assets := []models.Asset{}

	for _, c := range releaseAssets {
		if strings.Contains(c.GetName(), "sha256") {
			continue
		}
		cleanName := strings.ToLower(c.GetName())
		assetName := strings.Replace(c.GetName(), app.Name, "\" .. name .. \"", 1)
		assetName = strings.Replace(assetName, app.Version, "\" .. version .. \"", 1)
		path := "name"
		if !strings.Contains(cleanName, "tar") && !strings.Contains(cleanName, "zip") {
			path = strings.Replace(c.GetName(), app.Name, "name .. \"", 1) + "\""
		}

		if (strings.Contains(cleanName, "darwin") || strings.Contains(cleanName, "macos")) && strings.Contains(cleanName, app.Arch) {
			assets = append(assets, models.Asset{
				Arch:        "amd64",
				Os:          "darwin",
				AssertName:  assetName,
				InstallPath: "\"bin/\" .. name",
				Path:        path,
				Sha256:      checksumService.getChecksum(c.GetBrowserDownloadURL(), c.GetName()),
				Executable:  true,
			})

		}
		if strings.Contains(cleanName, "linux") && strings.Contains(cleanName, app.Arch) {
			assets = append(assets, models.Asset{
				Arch:        "amd64",
				Os:          "linux",
				AssertName:  assetName,
				InstallPath: "\"bin/\" .. name",
				Path:        path,
				Sha256:      checksumService.getChecksum(c.GetBrowserDownloadURL(), c.GetName()),
				Executable:  true,
			})
		}
		if strings.Contains(cleanName, "windows") && strings.Contains(cleanName, app.Arch) {
			if strings.Contains(cleanName, "tar") || strings.Contains(cleanName, "zip") {
				path = "name .. \".exe\""
			}
			if !strings.Contains(cleanName, "tar") && !strings.Contains(cleanName, "zip") && !strings.Contains(cleanName, "exe") {
				continue
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
