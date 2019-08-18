package gofishgithub

import (
	"fmt"
	"io/ioutil"

	"github.com/gofish-bot/gofish-bot/models"
)

func (p *GoFish) Lint(app *models.Application) error {

	bytes, err := ioutil.ReadFile("/usr/local/Fish/Rigs/github.com/fishworks/fish-food/Food/" + app.Name + ".lua")
	if err != nil {
		return err
	}

	food, err := p.GetAsFood(string(bytes))
	if err != nil {
		return err
	}

	errs := food.Lint()
	if len(errs) > 0 {
		e := ""
		for _, err := range errs {
			e += err.Error() + "\n"
		}
		return fmt.Errorf("Linting failed: \n%v", e)
	}
	p.Log.Debugf("Lint ok: %s", app.Name)
	return nil
}


func (p *GoFish) LintString(name, content string) error {

	food, err := p.GetAsFood(content)
	if err != nil {
		return err
	}

	errs := food.Lint()
	if len(errs) > 0 {
		e := ""
		for _, err := range errs {
			e += err.Error() + "\n"
		}
		return fmt.Errorf("Linting failed: %s \n - %v", name, e)
	}
	p.Log.Debugf("Lint ok: %s", name)
	return nil
}
