package module

import (
	"cmp"
	_ "embed"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wtsi-hgi/softpack/build"
)

//go:embed module.tmpl
var moduleTmplStr string
var moduleTmpl = template.Must(template.New("").Parse(moduleTmplStr))

func Install(moduleBase, installBase, envPath, envName, envVersion, description string, exes []string, pkgs []build.Package) (string, error) {
	if err := os.MkdirAll(filepath.Join(moduleBase, envPath), 0755); err != nil {
		return "", err
	}

	f, err := os.Create(filepath.Join(moduleBase, envPath, envVersion))
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	if err := cmp.Or(
		writeModuleFile(io.MultiWriter(f, &sb), installBase, envPath, envName, envVersion, description, exes, pkgs),
		f.Close(),
	); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func writeModuleFile(w io.Writer, installBase, envPath, envName, envVersion, description string, exes []string, pkgs []build.Package) error {
	return moduleTmpl.Execute(w, struct { //nolint:errcheck
		InstallDir         string
		EnvironmentPath    string
		EnvironmentName    string
		EnvironmentVersion string
		Exes               []string
		Description        []string
		Packages           []build.Package
	}{
		InstallDir:         installBase,
		EnvironmentPath:    envPath,
		EnvironmentName:    envName,
		EnvironmentVersion: envVersion,
		Exes:               exes,
		Description:        strings.Split(description, "\n"),
		Packages:           pkgs,
	})
}
