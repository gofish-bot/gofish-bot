package main

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/gofish-bot/gofish-bot/gofishgithub"
	"github.com/sirupsen/logrus"

	"github.com/gofish-bot/gofish-bot/log"

	"github.com/gofish-bot/gofish-bot/models"
	"github.com/gofish-bot/gofish-bot/strategy/generic"
	"github.com/gofish-bot/gofish-bot/strategy/github"
	"github.com/gofish-bot/gofish-bot/strategy/hashicorp"

	"github.com/go-yaml/yaml"
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

	var apply bool
	var verbose bool
	var target string

	app := cli.NewApp()
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "apply, a",
			Usage:       "Apply the planned actions",
			Destination: &apply,
		}, cli.StringFlag{
			Name:        "target",
			Usage:       "Target only one Food",
			Destination: &target,
		}, cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Full debug log",
			Destination: &verbose,
		},
	}

	app.Action = func(c *cli.Context) error {

		if verbose {
			log.L.Logger.SetLevel(logrus.DebugLevel)
		}

		ctx := context.Background()

		client := gofishgithub.CreateClient(ctx)
		goFish := &gofishgithub.GoFish{
			Client:      client,
			BotOrg:      "fmotrifork",
			FoodRepo:    "fish-food",
			FoodOrg:     "fishworks",
			AuthorName:  "Frederik Mogensen",
			AuthorEmail: "fmo@trifork.com",
		}
		// Hashicorp
		h := hashicorp.HashiCorp{GoFish: goFish}
		h.UpdateApplications(ctx, getApps("config/hashicorp.yaml", target), apply)

		// Generic
		gen := generic.Generic{GoFish: goFish}
		gen.UpdateApplications(ctx, getApps("config/generic.yaml", target), apply)

		// Github
		g := github.Github{GoFish: goFish}
		g.UpdateApplications(ctx, getApps("config/apps.yaml", target), apply)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.L.Fatal(err)
	}
}

func getApps(path, target string) []models.DesiredApp {

	c := []models.DesiredApp{}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.L.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.L.Fatalf("Unmarshal: %v", err)
	}
	if target == "" {
		return c
	}

	for _, app := range c {
		if app.Repo == target {
			return []models.DesiredApp{app}
		}
	}

	return []models.DesiredApp{}
}
