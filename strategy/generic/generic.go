package generic

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/gofishgithub"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/printer"

	"github.com/sirupsen/logrus"

	"strings"

	ghApi "github.com/google/go-github/v26/github"
)

type Generic struct {
	Log         *logrus.Logger
	Client      *ghApi.Client
	GithubToken string
	GoFish      *gofishgithub.GoFish
}

func (g *Generic) UpdateApplications(ctx context.Context, appsGithub []models.DesiredApp, createPullrequests bool) {
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

		checksumService := NewChecksumService(*app, g.GithubToken, g.Log)
		content, err := g.getUpgradedFood(ctx, app, checksumService)
		if err != nil {
			g.Log.Infof("Cound not upgrade current food: %s %s", app.Name, err)
			continue
		}

		err = g.GoFish.LintString(app.Name, content)
		if err != nil {
			g.Log.Warnf("Linting failed: %v", err)
			continue
		} else {
			g.Log.Infof("Linting ok: %v", app.Name)

		}
		if app.CurrentVersion != app.Version {
			missing := app.CurrentVersion == ""
			needsUpgrade := !missing && app.CurrentVersion != app.Version
			if needsUpgrade && createPullrequests {
				g.CreatePullRequest(ctx, app, content)
			} else if missing {
				g.Log.Infof("Generic strategy can not create new apps: %s", app.Name)
			}
		}
	}
}

func (g *Generic) CreateApplication(ctx context.Context, app models.DesiredApp) (*models.Application, error) {
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
		Version:      strings.Replace(releaseName, "v", "", 1),
		Arch:         app.Arch,
		Licence:      repoDetails.GetLicense().GetSPDXID(),
		Homepage:     homepage,
		Assets:       []models.Asset{},
	}

	return &application, nil
}

func (g *Generic) CreateLuaFile(application *models.Application, content string) {
	err := ioutil.WriteFile("/usr/local/Fish/Rigs/github.com/fishworks/fish-food/Food/"+application.Name+".lua", []byte(content), 0644)
	if err != nil {
		g.Log.Warn(err)
		return
	}
	return
}

func (g *Generic) CreatePullRequest(ctx context.Context, application *models.Application, content string) {

	g.Log.Infof("## Creating Pullrequest for %s version %s", application.Name, application.Version)
	err := g.GoFish.CreatePullRequest(ctx, application, []byte(content))
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

// The idea behind this strategy is based on the great work from
// https://github.com/karuppiah7890/uff/blob/master/food_utils.go

func (g *Generic) getUpgradedFood(ctx context.Context, app *models.Application, checksumService *ChecksumService) (string, error) {

	foodStr, _, err := g.GoFish.GetCurrentFood(ctx, app.Name)
	if err != nil {
		return "", fmt.Errorf("Cound not get current food: %s", app.Name)
	}

	versionUpgradedFoodStr := strings.ReplaceAll(string(foodStr), app.CurrentVersion, app.Version)
	versionUpgradedFood, err := g.GoFish.GetAsFood(versionUpgradedFoodStr)
	if err != nil {
		return "", fmt.Errorf("Cound not deserialize current food: %s", app.Name)
	}

	for _, foodPackage := range versionUpgradedFood.Packages {
		// u, _ := url.Parse(foodPackage.URL)
		ps := foodPackage.OS + "-" + foodPackage.Arch

		newSha := checksumService.getChecksum(foodPackage.URL, ps)

		g.Log.Debugf("Replacing old sha %s with %s", foodPackage.SHA256, newSha)
		versionUpgradedFoodStr = strings.ReplaceAll(versionUpgradedFoodStr, foodPackage.SHA256, newSha)
	}
	return versionUpgradedFoodStr, nil
}
