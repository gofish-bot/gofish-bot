package main

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/gofish-bot/gofish-bot/gofishgithub"

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

var log = *logrus.New()

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
			Name:        "apply",
			Usage:       "Apply the planned actions",
			Destination: &apply,
		}, cli.StringFlag{
			Name:        "project, p",
			Usage:       "github url",
			Destination: &githubPath,
		}, cli.StringFlag{
			Name:        "arch, a",
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

		if verbose {
			log.SetLevel(logrus.DebugLevel)
		}

		u, err := url.Parse(githubPath)
		if err != nil {
			panic(err)
		}

		githubToken, err := envy.MustGet("GITHUB_TOKEN")
		if err != nil {
			log.Fatalf("Error getting Github token: %v", err)
		}

		githubOrg, err := envy.MustGet("GITHUB_ORG")
		if err != nil {
			log.Fatalf("Error getting Github token: %v", err)
		}

		ctx := context.Background()
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
		log.Infof("%v", app)

		goFish := &gofishgithub.GoFish{
			Client:      client,
			BotOrg:      githubOrg,
			FoodRepo:    "fish-food",
			FoodOrg:     "fishworks",
			AuthorName:  "Frederik Mogensen",
			AuthorEmail: "fmo@trifork.com",
		}

		g := github.Github{
			Log:         &log,
			Client:      client,
			GithubToken: githubToken,
			GoFish:      goFish,
		}

		application, err := g.CreateApplication(ctx, app)
		if err != nil {
			log.Errorf("Error handling: %s", app.Repo)
		}

		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		columnFmt := color.New(color.FgYellow).SprintfFunc()

		tbl := table.New("Org", "Repo", "NewRelease")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

		tbl.AddRow(application.Organization, application.Name, application.Version)
		tbl.Print()

		g.CreateLuaFile(application)
		runLint(application)

		if apply {
			g.CreatePullRequest(ctx, application)
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func runLint(application *models.Application) {
	cmd := exec.Command("gofish", "lint", application.Name)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Error("linting failed")
	}
}
