package gofishgithub

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fishworks/gofish"
	"github.com/fishworks/gofish/pkg/home"
	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	"github.com/mholt/archiver/v3"
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

	f, err := p.GetAsFood(content)
	if err != nil {
		return fmt.Errorf("Converting to food (%s) failed: %v", name, err)
	}

	if len(f.Packages) < 3 {
		return fmt.Errorf("Linting failed: %s \n - %s", name, "Bad number of packages")
	}

	errs := f.Lint()
	if len(errs) > 0 {
		e := ""
		for _, err := range errs {
			e += err.Error() + "\n"
		}
		return fmt.Errorf("Linting failed: %s \n - '%v'", name, e)
	}
	log.L.Debugf("Lint ok: %s", name)

	for _, pkg := range f.Packages {
		u, err := url.Parse(pkg.URL)
		if err != nil {
			return fmt.Errorf("could not parse package URL '%s' as a URL: %v", pkg.URL, err)
		}

		cachedFilePath := filepath.Join(home.Cache(), fmt.Sprintf("%s-%s-%s-%s%s", f.Name, f.Version, pkg.OS, pkg.Arch, getExtension(u.Path)))
		fi, err := os.Stat(cachedFilePath)
		if err != nil {
			return err
		}
		// get the size
		size := fi.Size()
		if size < 100000 {
			return fmt.Errorf("Linting failed: %s \n - file %s is to small %s", name, cachedFilePath, humanize.Bytes(uint64(size)))
		}

		err = testInstall(f, pkg, cachedFilePath)
		if err != nil {
			return fmt.Errorf("Installing failed: %v", err)
		}
	}
	log.L.Debugf("Install ok: %s", name)

	return nil
}

func testInstall(f *gofish.Food, pkg *gofish.Package, src string) error {
	log.L.Debugf("Running install test")

	barrel := filepath.Join(home.Cache(), "barrel")
	barrelDir := filepath.Join(barrel, f.Name, f.Version, pkg.OS, pkg.Arch)

	u, err := url.Parse(pkg.URL)
	if err != nil {
		return fmt.Errorf("could not parse package URL '%s' as a URL: %v", pkg.URL, err)
	}

	err = os.RemoveAll(barrelDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(barrelDir, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	err = unarchiveOrCopy(src, barrelDir, u.Path)
	if err != nil {
		return fmt.Errorf("Linting failed: %s \n - Could not unarchive or copy: %s %v", f.Name, u.Path, err)
	}

	for _, r := range pkg.Resources {
		log.L.Debugf(" - Resource %s", r.Path)

		resourcePath := filepath.Join(barrelDir, r.Path)
		resourceFileInfo, err := os.Stat(resourcePath)
		if err != nil {
			return err
		}
		fType := "file"
		if resourceFileInfo.IsDir() {
			fType = "dir"
		}
		log.L.Debugf("%10s %7s %s %d bytes %s %s",
			pkg.OS, pkg.Arch,
			fType,
			resourceFileInfo.Size(),
			resourceFileInfo.ModTime().Format("2006-01-02 15:04"),
			resourceFileInfo.Name(),
		)
	}
	return nil
}

// From github.com/fishworks/gofish@v0.13.0/food.go
func unarchiveOrCopy(src, dest, urlPath string) error {

	// check and see if it can be unarchived by archiver
	if _, err := archiver.ByExtension(src); err == nil {
		return archiver.Unarchive(src, dest)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(filepath.Join(dest, filepath.Base(urlPath)))
	if err != nil {
		return fmt.Errorf("Creating out: %v", err)
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
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
