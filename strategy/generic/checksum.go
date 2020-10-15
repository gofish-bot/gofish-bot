package generic

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	ghApi "github.com/google/go-github/v32/github"

	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
)

type ChecksumService struct {
	application models.Application
	ghClient    *ghApi.Client
}

type Checksum struct {
	AssetName string
	SHA       string
}

func NewChecksumService(application models.Application, ghClient *ghApi.Client) *ChecksumService {
	c := &ChecksumService{
		application: application,
		ghClient:    ghClient,
	}
	return c

}

func (c *ChecksumService) getChecksum(url, assetName string) string {
	sha, err := c.getShaFromURL(assetName, url)
	if err != nil {
		log.L.Error(err)
		return ""
	}

	return sha
}

func (c *ChecksumService) getShaFromURL(assetName, assetURL string) (string, error) {
	content, err := c.downloadFile(assetName, assetURL)
	if err != nil {
		return "", fmt.Errorf("error while downloading package to calculate shasum: %v", err)
	}
	defer content.Close()

	h := sha256.New()
	if _, err := io.Copy(h, content); err != nil {
		return "", fmt.Errorf("error while calculating shasum of package: %v", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (c *ChecksumService) downloadFile(assetName, url string) (io.ReadCloser, error) {

	path := fmt.Sprintf("/tmp/gofish-bot/%s-%s-%s-%s%s", c.application.Organization, c.application.Name, c.application.ReleaseName, assetName, getExtension(url))

	if _, err := os.Stat(path); err == nil {
		log.L.Debugf("Getting from cache: %s", url)
		log.L.Debugf(" - path : %s", path)
		return getFile(path)
	}

	log.L.Debugf("Downloading: %s to %s", url, path)
	var req *http.Request
	var err error

	// Use GitHub client if asset is on github
	if strings.HasPrefix(url, "https://github.com/") {
		req, err = c.ghClient.NewRequest(http.MethodGet, url, nil)
	} else {
		req, err = http.NewRequest(http.MethodGet, url, nil)
	}
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	return getFile(path)
}

func getFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return f, nil
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
