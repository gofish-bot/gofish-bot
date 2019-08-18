package github

import (
	"io"
	"text/template"

	"github.com/gofish-bot/gofish-bot/models"
)

// https://github.com/fishworks/gofish/blob/master/cmd/gofish/create.go

const createTpl = `local name = "{{ .Name }}"
local org = "{{ .Organization }}"
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
                    path = {{$val.Path}},
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
	err := t.Execute(file, app)
	if err != nil {
		return err
	}
	return nil
}
