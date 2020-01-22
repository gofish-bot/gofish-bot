package main

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/gofish-bot/gofish-bot/gofishgithub"

	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/strategy/github"

	"github.com/sirupsen/logrus"

	"github.com/gobuffalo/envy"
	ghApi "github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli"
)

func main() {
	var verbose bool
	var githubPath string
	var arch string
	var path string
	var apply bool

	app := cli.NewApp()
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "apply, a",
			Usage:       "Apply the planned actions",
			Destination: &apply,
		}, cli.StringFlag{
			Name:        "project, p",
			Usage:       "github url",
			Destination: &githubPath,
			Required:    true,
		}, cli.StringFlag{
			Name:        "arch",
			Usage:       "Arch",
			Value:       "amd64",
			Destination: &arch,
		}, cli.StringFlag{
			Name:        "path",
			Usage:       "Path",
			Value:       "",
			Destination: &path,
		}, cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Full debug log",
			Destination: &verbose,
		},
	}

	app.Action = func(c *cli.Context) error {
		ctx := context.Background()

		if verbose {
			log.G(ctx).Logger.SetLevel(logrus.DebugLevel)
		}

		u, err := url.Parse(githubPath)
		if err != nil {
			panic(err)
		}

		githubToken, err := envy.MustGet("GITHUB_TOKEN")
		if err != nil {
			log.G(ctx).Fatalf("Error getting Github token: %v", err)
		}

		githubOrg, err := envy.MustGet("GITHUB_ORG")
		if err != nil {
			log.G(ctx).Fatalf("Error getting Github token: %v", err)
		}

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		client := ghApi.NewClient(tc)
		_ = client

		// Github
		app := models.DesiredApp{
			Repo: strings.Split(u.Path, "/")[2],
			Org:  strings.Split(u.Path, "/")[1],
			Arch: arch,
			Path: path,
		}
		log.G(ctx).Infof("%v", app)

		goFish := &gofishgithub.GoFish{
			Client:      client,
			BotOrg:      githubOrg,
			FoodRepo:    "fish-food",
			FoodOrg:     "fishworks",
			AuthorName:  "Frederik Mogensen",
			AuthorEmail: "fmo@trifork.com",
		}

		g := github.Github{
			GoFish: goFish,
		}

		application, err := g.CreateApplication(ctx, app)
		if err != nil {
			log.G(ctx).Errorf("Error handling: %s", app.Repo)
		}

		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		columnFmt := color.New(color.FgYellow).SprintfFunc()

		tbl := table.New("Org", "Repo", "NewRelease")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

		tbl.AddRow(application.Organization, application.Name, application.Version)
		tbl.Print()

		g.CreateLuaFile(ctx, application)
		runLint(application)

		err = g.GoFish.Lint(application)
		if err != nil {
			log.G(ctx).Warnf("Linting failed: %v", err)
		} else {
			log.G(ctx).Infof("Linting ok: %v", application.Name)
		}

		if apply {
			g.CreatePullRequest(ctx, application)
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.L.Fatal(err)
	}
}
func runLint(application *models.Application) {
	cmd := exec.Command("gofish", "lint", application.Name)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.L.Error("linting failed")
	}
}
