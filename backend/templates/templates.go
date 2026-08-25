package templates

import (
	_ "embed"
	"html/template"
)

//go:embed readme.tmpl
var readme string

//go:embed buildStatus.tmpl
var buildStatus string

//go:embed packageRequest.tmpl
var packageRequest string

var ReadmeTemplate *template.Template
var BuildStatusTemplate *template.Template
var PackageRequestTemplate *template.Template

func init() { //nolint:gochecknoinits
	ReadmeTemplate = template.Must(template.New("").Parse(readme))
	BuildStatusTemplate = template.Must(template.New("").Parse(buildStatus))
	PackageRequestTemplate = template.Must(template.New("").Parse(packageRequest))
}
