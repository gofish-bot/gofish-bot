package gofishgithub

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fishworks/gofish/pkg/home"
	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
)

func (p *GoFish) Lint(app *models.Application) error {

	bytes, err := ioutil.ReadFile("/usr/local/gofish/tmp/github.com/fmotrifork/fish-food/Food/" + app.Name + ".lua")
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
		return fmt.Errorf("Converting to food (%s) failed: %v", name, err)
	}

	if len(food.Packages) < 3 {
		return fmt.Errorf("Linting failed: %s \n - %s", name, "Bad number of packages")
	}

	errs := food.Lint()
	if len(errs) > 0 {
		e := ""
		for _, err := range errs {
			e += err.Error() + "\n"
		}
		return fmt.Errorf("Linting failed: %s \n - '%v'", name, e)
	}
	log.L.Debugf("Lint ok: %s", name)

	for _, pkg := range food.Packages {
		u, err := url.Parse(pkg.URL)
		if err != nil {
			return fmt.Errorf("could not parse package URL '%s' as a URL: %v", pkg.URL, err)
		}

		cachedFilePath := filepath.Join(home.Cache(), fmt.Sprintf("%s-%s-%s-%s%s", food.Name, food.Version, pkg.OS, pkg.Arch, getExtension(u.Path)))
		fi, err := os.Stat(cachedFilePath)
		if err != nil {
			return err
		}
		// get the size
		size := fi.Size()
		if size < 100000 {
			return fmt.Errorf("Linting failed: %s \n - file %s is to small %s", name, cachedFilePath, humanize.Bytes(uint64(size)))
		}
	}

	return nil
}

// From github.com/fishworks/gofish@v0.13.0/food.go
func getExtension(path string) string {
	urlParts := strings.Split(path, "/")
	parts := strings.Split(urlParts[len(urlParts)-1], ".")
	if len(parts) < 2 {
		return filepath.Ext(path)
	}
	return "." + strings.Join([]string{parts[len(parts)-2], parts[len(parts)-1]}, ".")
}
