package gofishgithub

import (
	"context"
	"fmt"

	"github.com/gofish-bot/gofish-bot/models"
	"github.com/google/go-github/v32/github"

	"github.com/fishworks/gofish"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

func (p *GoFish) GetCurrentVersion(ctx context.Context, app models.DesiredApp) (string, error) {
	return p.getVersion(ctx, app, "main")
}

func (p *GoFish) getVersion(ctx context.Context, app models.DesiredApp, ref string) (string, error) {

	food, err := p.getFood(ctx, app.Name, ref)
	if err != nil {
		return "", err
	}

	return food.Version, nil
}

func (p *GoFish) GetCurrentFood(ctx context.Context, appRepo string) (string, *gofish.Food, error) {
	content, err := p.getContent(ctx, appRepo, "main")
	if err != nil {
		return "", nil, err
	}
	food, err := p.getFood(ctx, appRepo, "main")
	if err != nil {
		return "", nil, err
	}
	return content, food, nil
}
func (p *GoFish) GetAsFood(content string) (*gofish.Food, error) {
	l := lua.NewState()
	defer l.Close()
	if err := l.DoString(content); err != nil {
		return nil, err
	}
	var food gofish.Food
	if err := gluamapper.Map(l.GetGlobal("food").(*lua.LTable), &food); err != nil {
		return nil, err
	}
	return &food, nil
}

func (p *GoFish) getFood(ctx context.Context, appName string, ref string) (*gofish.Food, error) {
	content, err := p.getContent(ctx, appName, ref)
	if err != nil {
		return nil, err
	}

	l := lua.NewState()
	defer l.Close()
	if err := l.DoString(content); err != nil {
		return nil, err
	}
	var food gofish.Food
	if err := gluamapper.Map(l.GetGlobal("food").(*lua.LTable), &food); err != nil {
		return nil, err
	}
	return &food, nil
}

func (p *GoFish) getContent(ctx context.Context, appName string, ref string) (string, error) {

	getOpts := &github.RepositoryContentGetOptions{Ref: ref}
	res, _, _, err := p.Client.Repositories.GetContents(ctx, p.FoodOrg, p.FoodRepo, fmt.Sprintf("Food/%s.lua", appName), getOpts)
	if err != nil {
		return "", err
	}

	content, err := res.GetContent()
	if err != nil {
		return "", err
	}

	return content, nil
}
