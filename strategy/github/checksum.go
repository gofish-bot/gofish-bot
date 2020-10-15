package github

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gofish-bot/gofish-bot/log"
	"github.com/gofish-bot/gofish-bot/models"
	ghApi "github.com/google/go-github/v32/github"
)

type ChecksumService struct {
	application models.Application
	checksums   []Checksum
	preloaded   bool
	ghClient    *ghApi.Client
}

type Checksum struct {
	AssetName string
	SHA       string
}

func NewChecksumService(application models.Application, ghClient *ghApi.Client, assets []*ghApi.ReleaseAsset) *ChecksumService {
	c := &ChecksumService{
		application: application,
		ghClient:    ghClient,
	}
	c.preLoadFromAssets(assets)
	return c

}

func (c *ChecksumService) getChecksum(url, assetName string) string {

	for _, checksum := range c.checksums {
		if strings.Contains(checksum.AssetName, assetName) {
			log.L.Debugf("Found sha %s for %s in %s\n", checksum.SHA, assetName, checksum.AssetName)
			return checksum.SHA
		}
	}
	log.L.Debugf("Falling back to calculating SHA for %s using %s\n", assetName, url)
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

func (c *ChecksumService) preLoadFromAssets(assets []*ghApi.ReleaseAsset) {
	checksums := ""

	for _, asset := range assets {
		if strings.Contains(asset.GetName(), "checksums") && !strings.Contains(asset.GetName(), "512") {
			reader, err := c.downloadFile(asset.GetName(), asset.GetBrowserDownloadURL())
			if err != nil {
				log.L.Errorf("Could not download checksums: %s %v", asset.GetBrowserDownloadURL(), err)
			}
			defer reader.Close()
			checksumBytes, err := ioutil.ReadAll(reader)
			if err != nil {
				log.L.Errorf("Could not download checksums: %v", err)
			}
			checksums = string(checksumBytes)

		}
		if strings.Contains(asset.GetName(), "sha256") {
			csReader, err := c.downloadFile(asset.GetName(), asset.GetBrowserDownloadURL())
			if err != nil {
				log.L.Errorf("Could not download checksums: %v", csReader)
			}
			csBytes, err := ioutil.ReadAll(csReader)
			if err != nil {
				log.L.Errorf("Could not download checksums: %v", err)
			}
			csStr := string(csBytes)

			// csStr may be either "sha assetname" or just "sha"
			// - If we postfix with assetname then the second element will always contain the assetname
			checksums += fmt.Sprintf("%s %s\n", strings.Trim(csStr, "\n"), strings.ReplaceAll(asset.GetName(), ".sha256", ""))
		}
	}
	cs := []Checksum{}

	for _, line := range strings.Split(strings.TrimSuffix(checksums, "\n"), "\n") {
		if line == "" {
			continue
		}
		x := strings.Fields(strings.TrimSpace(line))
		log.L.Debugf("SHA line: %s len: %d", x, len(x))

		if len(x) < 2 {
			continue
		}
		cs = append(cs, Checksum{
			SHA:       x[0],
			AssetName: x[1],
		})
	}
	c.checksums = cs
}

func (c *ChecksumService) downloadFile(assetName, url string) (io.ReadCloser, error) {

	path := fmt.Sprintf("/tmp/gofish-bot/%s-%s-%s-%s", c.application.Organization, c.application.Name, c.application.ReleaseName, assetName)

	if _, err := os.Stat(path); err == nil {
		log.L.Debugf("Getting from cache: %s", url)
		return getFile(path)
	}

	log.L.Debugf("Downloading: %s to %s", url, path)
	req, err := c.ghClient.NewRequest(http.MethodGet, url, nil)
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
