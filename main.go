package main

import (
	"context"
	"io/ioutil"
	"os"
	"path"

	"github.com/fishworks/gofish/pkg/home"
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
	mkDir()
}

func mkDir() {
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err == nil || os.IsExist(err) {
		return
	} else {
		panic(err)
	}
}

func main() {

	var clean bool
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
		}, cli.BoolFlag{
			Name:        "clean, c",
			Usage:       "Clear all cached packages",
			Destination: &clean,
		},
	}

	app.Action = func(c *cli.Context) error {

		if verbose {
			log.L.Logger.SetLevel(logrus.DebugLevel)
		}

		if clean {
			clearDir(tmpDir)
			clearDir(home.Cache())
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

	for i, app := range c {
		if app.Name == "" {
			app.Name = app.Repo
			c[i] = app
		}
	}

	if target == "" {
		return c
	}

	for _, app := range c {
		if app.Name == target {
			return []models.DesiredApp{app}
		}
	}

	return []models.DesiredApp{}
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
