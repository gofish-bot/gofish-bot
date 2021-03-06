package github

import (
	"io"
	"text/template"

	"github.com/gofish-bot/gofish-bot/models"
)

// https://github.com/fishworks/gofish/blob/master/cmd/gofish/create.go

const createTplOtherName = `local name = "{{ .Name }}"
local release = "{{ .ReleaseName }}"
local version = "{{ .Version }}"
food = {
    name = name,
    description = "{{ .Description }}",
    license = "{{ .Licence }}",
    homepage = "{{ .Homepage }}",
    version = version,
    packages = {
        {{- range $index, $val := .Assets}}{{if $index}},{{end}}
        {
            os = "{{$val.Os}}",
            arch = "{{$val.Arch}}",
            url = "https://github.com/{{ $.Organization }}/{{ $.Repo }}/releases/download/" .. release .. "/{{$val.AssertName}}",
            sha256 = "{{$val.Sha256}}",
            resources = {
                {
                    {{- if $.Path }}
                    path = "{{ $.Path }}" .. {{$val.Path}},
                    {{- else }}
                    path = {{$val.Path}},
                    {{- end }}
                    installpath = {{$val.InstallPath}}{{- if $val.Executable}},
                    executable = true
                    {{- end}}
                }
            }
        }{{- end}}
    }
}
`

const createTpl = `local name = "{{ .Name }}"
local release = "{{ .ReleaseName }}"
local version = "{{ .Version }}"
food = {
    name = name,
    description = "{{ .Description }}",
    license = "{{ .Licence }}",
    homepage = "{{ .Homepage }}",
    version = version,
    packages = {
        {{- range $index, $val := .Assets}}{{if $index}},{{end}}
        {
            os = "{{$val.Os}}",
            arch = "{{$val.Arch}}",
            url = "https://github.com/{{ $.Organization }}/" .. name .. "/releases/download/" .. release .. "/{{$val.AssertName}}",
            sha256 = "{{$val.Sha256}}",
            resources = {
                {
                    {{- if $.Path }}
                    path = "{{ $.Path }}" .. {{$val.Path}},
                    {{- else }}
                    path = {{$val.Path}},
                    {{- end }}
                    installpath = {{$val.InstallPath}}{{- if $val.Executable}},
                    executable = true
                    {{- end}}
                }
            }
        }{{- end}}
    }
}
`

func serializeLuaContent(app *models.Application, file io.Writer) error {
	t := template.Must(template.New("create").Parse(createTpl))

	if app.Name != app.Repo {
		t = template.Must(template.New("create").Parse(createTplOtherName))
	}
	err := t.Execute(file, app)
	if err != nil {
		return err
	}
	return nil
}
