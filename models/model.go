package models

type DesiredApp struct {
	Repo           string
	Org            string
	Arch           string
	Onlyprerelease bool
	Path        string
}

type Asset struct {
	Arch        string
	Os          string
	AssertName  string
	InstallPath string
	Path        string
	Sha256      string
	Executable  bool
}

type Application struct {
	ReleaseName    string
	Name           string
	Organization   string
	Path           string
	CurrentVersion string
	Version        string
	Arch           string
	Description    string
	Licence        string
	Homepage       string
	Assets         []Asset
}
