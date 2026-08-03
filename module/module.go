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

func Install(moduleBase, installBase, envPath, envName, envVer, description string, exes []string, pkgs []build.Package) error {
	if err := os.MkdirAll(filepath.Join(moduleBase, envPath, envName), 0755); err != nil {
		return err
	}

	f, err := os.Create(ModuleFile(moduleBase, envPath, envName, envVer))
	if err != nil {
		return err
	}

	if err := cmp.Or(
		writeModuleFile(f, installBase, envPath, envName, envVer, description, exes, pkgs),
		f.Close(),
	); err != nil {
		return err
	}

	return nil
}

func ModuleFile(moduleBase, envPath, envName, envVersion string) string {
	return filepath.Join(moduleBase, envPath, envName, envVersion)
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
