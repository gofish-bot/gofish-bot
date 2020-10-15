package gofishgithub

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/gobuffalo/envy"
	"golang.org/x/oauth2"

	"github.com/pkg/errors"

	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/google/go-github/v32/github"
	ghApi "github.com/google/go-github/v32/github"
)

type GoFish struct {
	Client      *ghApi.Client
	BotOrg      string
	FoodRepo    string
	FoodOrg     string
	AuthorName  string
	AuthorEmail string
}

func CreateClient(ctx context.Context) *ghApi.Client {
	githubToken, err := envy.MustGet("GITHUB_TOKEN")
	if err != nil {
		log.G(ctx).Fatalf("Error getting Github token: %v", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	return ghApi.NewClient(tc)
}

// https://godoc.org/github.com/google/go-github/github#example-RepositoriesService-CreateFile
func (p *GoFish) CreatePullRequest(ctx context.Context, application *models.Application, fileContent []byte) error {
	branch := fmt.Sprintf("%s-%s", application.Name, application.ReleaseName)
	var err error

	err = p.createNewBranch(ctx, application, branch)
	if err != nil {
		return err
	}
	body := fmt.Sprintf("Updating package %s to release %s.", application.Name, application.ReleaseName)
	if application.ReleaseDescription != "" {
		body = fmt.Sprintf("Updating package %s to release %s. \n\n# Release info \n\n %s", application.Name, application.ReleaseName, application.ReleaseDescription)
	}
	if application.CurrentVersion != "" {
		err = p.updateFile(ctx, application, fileContent, branch)
		if err != nil {
			return err
		}
	} else {
		body = fmt.Sprintf("Creating package %s in version %s.", application.Name, application.ReleaseName)

		err = p.createFile(ctx, application, fileContent, branch)
		if err != nil {
			return err
		}
	}
	err = p.newPullRequest(ctx, application, branch, body)
	if err != nil {
		return err
	}
	return nil
}

func (p *GoFish) updateFile(ctx context.Context, application *models.Application, fileContent []byte, branch string) error {
	log.G(ctx).Debug("Updating File")

	getOpts := &github.RepositoryContentGetOptions{Ref: branch}
	res, _, _, err := p.Client.Repositories.GetContents(ctx, p.BotOrg, p.FoodRepo, fmt.Sprintf("Food/%s.lua", application.Name), getOpts)
	if err != nil {
		return err
	}

	opts := &github.RepositoryContentFileOptions{
		Message:   github.String(fmt.Sprintf("%s %s", application.Name, application.Version)),
		Content:   fileContent,
		Branch:    github.String(branch),
		Author:    &github.CommitAuthor{Name: github.String(p.AuthorName), Email: github.String(p.AuthorEmail)},
		Committer: &github.CommitAuthor{Name: github.String(p.AuthorName), Email: github.String(p.AuthorEmail)},
		SHA:       github.String(res.GetSHA()),
	}
	_, _, err = p.Client.Repositories.UpdateFile(ctx, p.BotOrg, p.FoodRepo, fmt.Sprintf("Food/%s.lua", application.Name), opts)
	if err != nil {
		return err
	}

	return nil
}

func (p *GoFish) createFile(ctx context.Context, application *models.Application, fileContent []byte, branch string) error {
	log.G(ctx).Debug("Creating new File")

	opts := &github.RepositoryContentFileOptions{
		Message:   github.String(fmt.Sprintf("%s %s", application.Name, application.Version)),
		Content:   fileContent,
		Branch:    github.String(branch),
		Author:    &github.CommitAuthor{Name: github.String(p.AuthorName), Email: github.String(p.AuthorEmail)},
		Committer: &github.CommitAuthor{Name: github.String(p.AuthorName), Email: github.String(p.AuthorEmail)},
	}
	_, _, err := p.Client.Repositories.UpdateFile(ctx, p.BotOrg, p.FoodRepo, fmt.Sprintf("Food/%s.lua", application.Name), opts)
	if err != nil {
		return err
	}

	return nil
}

func (p *GoFish) createNewBranch(ctx context.Context, application *models.Application, branch string) error {
	log.G(ctx).Debugf("Creating new Branch %s\n", branch)

	ref, _, err := p.Client.Git.GetRef(ctx, p.BotOrg, p.FoodRepo, "refs/heads/main")
	if err != nil {
		return errors.Wrapf(err, "Error creating ref %s", ref)
	}

	_, _, err = p.Client.Git.CreateRef(ctx, p.BotOrg, p.FoodRepo, &github.Reference{
		Ref:    github.String(fmt.Sprintf("refs/heads/%s", branch)),
		Object: ref.GetObject(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *GoFish) newPullRequest(ctx context.Context, application *models.Application, branch, body string) error {
	log.G(ctx).Debugf("Sending pull request\n")
	newPR := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("%s %s", application.Name, application.Version)),
		Head:                github.String(p.BotOrg + ":" + branch),
		Base:                github.String("main"),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}

	pr, res, err := p.Client.PullRequests.Create(context.Background(), p.FoodOrg, p.FoodRepo, newPR)
	if err != nil {
		r, _ := ioutil.ReadAll(res.Response.Body)
		defer res.Response.Body.Close()
		log.G(ctx).Warnf("%s", r)
		return err
	}

	log.G(ctx).Infof("PR created: %s\n", pr.GetHTMLURL())
	return nil
}
