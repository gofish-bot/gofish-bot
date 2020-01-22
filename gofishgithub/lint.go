package gofishgithub

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fishworks/gofish"

	"github.com/dustin/go-humanize"
	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
)

func (p *GoFish) Lint(app *models.Application) error {

	bytes, err := ioutil.ReadFile("/usr/local/Fish/Rigs/github.com/fishworks/fish-food/Food/" + app.Name + ".lua")
	if err != nil {
		return err
	}

	return p.lint(app.Name, string(bytes))
}

func (p *GoFish) LintString(name, content string) error {

	return p.lint(name, content)
}

func (p *GoFish) lint(name, content string) error {

	food, err := p.GetAsFood(content)
	if err != nil {
		return err
	}

	if len(food.Packages) != 3 {
		return fmt.Errorf("Linting failed: %s \n - %s", name, "Bad number of packages")
	}

	errs := food.Lint()
	if len(errs) > 0 {
		e := ""
		for _, err := range errs {
			e += err.Error() + "\n"
		}
		return fmt.Errorf("Linting failed: %s \n - %v", name, e)
	}
	log.L.Debugf("Lint ok: %s", name)

	for _, pkg := range food.Packages {
		u, err := url.Parse(pkg.URL)
		if err != nil {
			return fmt.Errorf("could not parse package URL '%s' as a URL: %v", pkg.URL, err)
		}

		cachedFilePath := filepath.Join(gofish.UserHome(gofish.UserHomePath).Cache(), fmt.Sprintf("%s-%s-%s-%s%s", food.Name, food.Version, pkg.OS, pkg.Arch, filepath.Ext(u.Path)))
		fi, err := os.Stat(cachedFilePath)
		if err != nil {
			return err
		}
		// get the size
		size := fi.Size()
		if size < 1000000 {
			return fmt.Errorf("Linting failed: %s \n - file %s is to small %s", name, cachedFilePath, humanize.Bytes(uint64(size)))
		}
	}

	return nil
}
