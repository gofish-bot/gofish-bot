package hashicorp

import (
	"io"
	"text/template"

	"github.com/gofish-bot/gofish-bot/models"
)

const createHashicorpTpl = `local name = "{{ .Name }}"
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
            url = "https://releases.hashicorp.com/" .. name .. "/" .. version .. "/" .. name .. "_" .. version .. "{{$val.AssertName}}",
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

func newCreateHashicorp(app *models.Application, file io.Writer) error {

	t := template.Must(template.New("create").Parse(createHashicorpTpl))
	err := t.Execute(file, app)
	if err != nil {
		return err
	}
	return nil
}
