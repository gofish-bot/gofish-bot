package main

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/fishworks/gofish/pkg/home"
	"github.com/gofish-bot/gofish-bot/gofishgithub"

	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/strategy/github"

	"github.com/sirupsen/logrus"

	"github.com/gobuffalo/envy"
	ghApi "github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/urfave/cli"
)

const tmpDir = "/tmp/gofish-bot"

func init() {
	mkDir(tmpDir)
}

func mkDir(path string) {
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err == nil || os.IsExist(err) {
		return
	} else {
		panic(err)
	}
}

func main() {
	var clean bool
	var verbose bool
	var githubPath string
	var name string
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
			Name:        "name, n",
			Usage:       "alternative name for the food",
			Destination: &name,
			Required:    false,
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
		}, cli.BoolFlag{
			Name:        "clean, c",
			Usage:       "Clear all cached packages",
			Destination: &clean,
		},
	}

	app.Action = func(c *cli.Context) error {
		ctx := context.Background()

		if verbose {
			log.G(ctx).Logger.SetLevel(logrus.DebugLevel)
		}

		if clean {
			clearDir(tmpDir)
			clearDir(home.Cache())
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
		org := strings.Split(u.Path, "/")[1]
		repo := strings.Split(u.Path, "/")[2]

		if name == "" {
			name = repo
		}

		// Github
		app := models.DesiredApp{
			Name: name,
			Repo: repo,
			Org:  org,
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
			log.G(ctx).Errorf("Error handling: %s", app.Name)
		}

		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		columnFmt := color.New(color.FgYellow).SprintfFunc()

		tbl := table.New("Org", "Repo", "NewRelease")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

		tbl.AddRow(application.Organization, application.Name, application.Version)
		tbl.Print()

		g.CreateLuaFile(ctx, application)

		err = g.GoFish.Lint(application)
		if err != nil {
			log.G(ctx).Warnf("Linting failed in cmd: %v", err)
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

func clearDir(dir string) error {
	log.L.Debugf("Cleaning: %s", dir)
	names, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entery := range names {
		file := path.Join([]string{dir, entery.Name()}...)
		log.L.Debugf(" - deleting: %s", file)
		os.RemoveAll(file)
	}
	return nil
}
