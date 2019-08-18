package gofishgithub

import (
	"github.com/sirupsen/logrus"
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/google/go-github/v26/github"
	ghApi "github.com/google/go-github/v26/github"
	"github.com/gofish-bot/gofish-bot/models"
)

type GoFish struct {
	Client      *ghApi.Client
	BotOrg      string
	FoodRepo    string
	FoodOrg     string
	AuthorName  string
	AuthorEmail string
	Log 		*logrus.Logger
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
	p.Log.Debug("Updating File")

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
	p.Log.Debug("Creating new File")

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
	p.Log.Debugf("Creating new Branch %s\n", branch)

	ref, _, err := p.Client.Git.GetRef(ctx, p.BotOrg, p.FoodRepo, "refs/heads/master")
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
	p.Log.Debugf("Sending pull request\n")
	newPR := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("%s %s", application.Name, application.Version)),
		Head:                github.String(p.BotOrg + ":" + branch),
		Base:                github.String("master"),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := p.Client.PullRequests.Create(context.Background(), p.FoodOrg, p.FoodRepo, newPR)
	if err != nil {
		return err
	}

	p.Log.Infof("PR created: %s\n", pr.GetHTMLURL())
	return nil
}
