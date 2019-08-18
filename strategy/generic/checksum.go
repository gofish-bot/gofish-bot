package generic

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/gofish-bot/gofish-bot/models"
)

type ChecksumService struct {
	Log         *logrus.Logger
	application models.Application
	githubToken string
}

type Checksum struct {
	AssetName string
	SHA       string
}

func NewChecksumService(application models.Application, githubToken string, log *logrus.Logger) *ChecksumService {
	c := &ChecksumService{
		Log:         log,
		application: application,
		githubToken: githubToken,
	}
	return c

}

func (c *ChecksumService) getChecksum(url, assetName string) string {
	sha, err := c.getShaFromURL(assetName, url)
	if err != nil {
		c.Log.Error(err)
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

	path := fmt.Sprintf("/tmp/gofish-bot/%s-%s-%s-%s", c.application.Organization, c.application.Name, c.application.ReleaseName, assetName)

	if _, err := os.Stat(path); err == nil {
		c.Log.Debugf("Getting from cache: %s", url)
		return getFile(path)
	}

	c.Log.Debugf("Downloading: %s to %s", url, path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Only add auth header if asset is on github
	if strings.HasPrefix(url, "https://github.com/") {
		req.Header.Set("Authorization", "token "+c.githubToken)
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
