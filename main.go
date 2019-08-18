package main

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/gofish-bot/gofish-bot/gofishgithub"

	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/strategy/generic"
	"github.com/gofish-bot/gofish-bot/strategy/github"
	"github.com/gofish-bot/gofish-bot/strategy/hashicorp"

	"github.com/sirupsen/logrus"

	"github.com/go-yaml/yaml"
	"github.com/gobuffalo/envy"
	ghApi "github.com/google/go-github/v26/github"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
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

var log = *logrus.New()

func main() {

	var apply bool
	var verbose bool

	app := cli.NewApp()
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "apply",
			Usage:       "Apply the planned actions",
			Destination: &apply,
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

		githubToken, err := envy.MustGet("GITHUB_TOKEN")
		if err != nil {
			log.Fatalf("Error getting Github token: %v", err)
		}

		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := ghApi.NewClient(tc)

		goFish := &gofishgithub.GoFish{
			Client:      client,
			BotOrg:      "gofish-bot",
			FoodRepo:    "fish-food",
			FoodOrg:     "fishworks",
			AuthorName:  "GoFish Bot",
			AuthorEmail: "GoFishBot@gmail.com",
			Log:         &log,
		}
		// Hashicorp
		h := hashicorp.HashiCorp{Client: client, GithubToken: githubToken, GoFish: goFish, Log: &log}
		h.UpdateApplications(ctx, getApps("config/hashicorp.yaml"), apply)

		// Generic
		gen := generic.Generic{Client: client, GithubToken: githubToken, GoFish: goFish, Log: &log}
		gen.UpdateApplications(ctx, getApps("config/generic.yaml"), apply)

		// Github
		g := github.Github{Client: client, GithubToken: githubToken, GoFish: goFish, Log: &log}
		g.UpdateApplications(ctx, getApps("config/apps.yaml"), apply)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getApps(path string) []models.DesiredApp {

	c := []models.DesiredApp{}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
