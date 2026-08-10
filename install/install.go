package install

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/wtsi-hgi/softpack/artefacts"
	"github.com/wtsi-hgi/softpack/build"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/module"
)

// Install builds an environment, installs the module file, and copies the build
// artefacts to the given artefactBase location.
func Install(c *config.Config, e db.Environment) (b *build.Artefacts, err error) {
	envVer := strconv.Itoa(e.Version)
	installPath := filepath.Join(c.InstallDir, e.Path, e.Name, envVer+"-scripts")

	if err = os.MkdirAll(installPath, 0755); err != nil { //nolint:mnd
		return nil, err
	}

	arts, err := build.Build(
		c.BaseImgPath, c.TempDir, installPath, c.WrapperScript, c.AptSrc, e.Packages,
	)
	if err != nil {
		return arts, err
	}

	if err = module.Install(
		c.ModulePath, c.InstallDir,
		e.Path, e.Name, envVer, e.Description, arts.Exes, arts.Packages,
	); err != nil {
		return arts, err
	}

	if err := artefacts.Store(c.ArtefactStore, filepath.Join(e.Path, e.Name, envVer), map[string]artefacts.Opener{
		"module":          artefacts.File(module.ModuleFile(c.ModulePath, e.Path, e.Name, envVer)),
		"singularity.sif": artefacts.File(build.SingularityPath(installPath)),
		"build.log":       artefacts.Data(arts.Log),
	}); err != nil {
		return arts, err
	}

	return arts, nil
}
