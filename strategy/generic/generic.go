package generic

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/blang/semver"
	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/gofishgithub"
	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/printer"

	"strings"

	ghApi "github.com/google/go-github/v32/github"
)

type Generic struct {
	GoFish *gofishgithub.GoFish
}

func (g *Generic) UpdateApplications(ctx context.Context, appsGithub []models.DesiredApp, createPullrequests bool) {
	applications := []*models.Application{}

	for _, app := range appsGithub {
		application, err := g.CreateApplication(ctx, app)
		if err != nil {
			log.G(ctx).Warnf("Error in handling %s: %v", app.Name, err)
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

			checksumService := NewChecksumService(*app, g.GoFish.Client)
			content, err := g.getUpgradedFood(ctx, app, checksumService)
			if err != nil {
				log.G(ctx).Infof("Cound not upgrade current food: %s %s", app.Name, err)
				continue
			}
			missing := app.CurrentVersion == ""
			needsUpgrade := !missing && app.CurrentVersion != app.Version
			upgradeToBeta := (!strings.Contains(app.CurrentVersion, "beta")) && strings.Contains(app.Version, "beta")

			if needsUpgrade {
				g.CreateLuaFile(ctx, app, content)
				err = g.GoFish.LintString(app.Name, content)
				if err != nil {

					// Allow generic packages to have fewer packages
					if strings.Contains(err.Error(), "Bad number of packages") {
						log.G(ctx).Infof("Linting failed, but continuing: %v", err)
					} else {
						log.G(ctx).Warnf("Linting failed: %v", err)
						continue
					}
				} else {
					log.G(ctx).Infof("Linting ok: %v", app.Name)
				}
			}

			if upgradeToBeta {
				log.G(ctx).Infof("Will not upgrade to beta release: %s", app.Name)
			} else if needsUpgrade && createPullrequests {
				log.G(ctx).Infof("Creating pr for release: %s", app.Name)
				g.CreateLuaFile(ctx, app, content)
				g.CreatePullRequest(ctx, app, content)
			} else if missing {
				log.G(ctx).Infof("Generic strategy can not create new apps: %s", app.Name)
			}
		}
	}
}

func (g *Generic) CreateApplication(ctx context.Context, app models.DesiredApp) (*models.Application, error) {
	log.G(ctx).Infof("## Creating Application for %s: https://github.com/%s/%s/releases/", app.Name, app.Org, app.Repo)

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
		ReleaseName:        releaseName,
		ReleaseDescription: release.GetBody(),
		ReleaseLink:        release.GetHTMLURL(),
		Name:               app.Name,
		Repo:               app.Repo,
		Description:        repoDetails.GetDescription(),
		Organization:       app.Org,
		Version:            strings.Replace(releaseName, "v", "", 1),
		Arch:               app.Arch,
		Licence:            repoDetails.GetLicense().GetSPDXID(),
		Homepage:           homepage,
		Assets:             []models.Asset{},
	}

	return &application, nil
}

func (g *Generic) CreateLuaFile(ctx context.Context, application *models.Application, content string) {
	err := ioutil.WriteFile("/usr/local/gofish/tmp/github.com/fmotrifork/fish-food/Food/"+application.Name+".lua", []byte(content), 0644)
	if err != nil {
		log.G(ctx).Warn(err)
	}
}

func (g *Generic) CreatePullRequest(ctx context.Context, application *models.Application, content string) {

	log.G(ctx).Infof("## Creating Pullrequest for %s version %s", application.Name, application.Version)
	err := g.GoFish.CreatePullRequest(ctx, application, []byte(content))
	if err != nil {
		log.G(ctx).Warnf("Failed creating PR: %v", err)
		return
	}
}

func findRelease(app models.DesiredApp, releaseList []*ghApi.RepositoryRelease) *ghApi.RepositoryRelease {

	var release *ghApi.RepositoryRelease
	newestRelease, _ := semver.Make("0.0.0")

	for _, v := range releaseList {
		releaseVersion, err := semver.Make(strings.Replace(v.GetTagName(), "v", "", 1))
		if err != nil {
			continue
		}

		if releaseVersion.GT(newestRelease) && len(releaseVersion.Pre) == 0 && !strings.Contains(v.GetTagName(), "edge") {
			newestRelease = releaseVersion
			release = v
		}
	}

	if app.Onlyprerelease {
		for k, v := range releaseList {
			if v.GetPrerelease() {
				continue
			}
			return releaseList[k]
		}
	}
	if release == nil {
		log.L.Warnf("Falling back to first release in list: %v", releaseList[0].GetTagName())
		return releaseList[0]
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

		log.G(ctx).Debugf("Replacing old sha %s with %s", foodPackage.SHA256, newSha)
		versionUpgradedFoodStr = strings.ReplaceAll(versionUpgradedFoodStr, foodPackage.SHA256, newSha)
	}
	return versionUpgradedFoodStr, nil
}
