package install

import (
	"os"
	"path/filepath"

	"github.com/wtsi-hgi/softpack/artefacts"
	"github.com/wtsi-hgi/softpack/build"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/module"
)

// Install builds an environment, installs the module file, and copies the build
// artefacts to the given artefactBase location.
func Install(c *config.Config, e db.Environment) (b *build.Artefacts, err error) {
	installPath := filepath.Join(c.InstallDir, e.Path, e.Name, e.Version+"-scripts")

	if err = os.MkdirAll(installPath, 0755); err != nil { //nolint:mnd
		return &build.Artefacts{}, err
	}

	arts, err := build.Build(
		c.BaseImgPath, c.TempDir, installPath, c.WrapperScript, c.AptSrc, e.Packages,
	)
	if err != nil { // build log
		return arts, err
	}

	if err = module.Install(
		c.ModulePath, c.InstallDir,
		e.Path, e.Name, e.Version, e.Description, arts.Exes, arts.Packages,
	); err != nil {
		return arts, err
	}

	if err := artefacts.Store(c.ArtefactStore, filepath.Join(e.Path, e.Name, e.Version), map[string]artefacts.Opener{
		"module":          artefacts.File(module.ModuleFile(c.ModulePath, e.Path, e.Name, e.Version)),
		"singularity.sif": artefacts.File(build.SingularityPath(installPath)),
		"build.log":       artefacts.Data(arts.Log),
	}); err != nil {
		return arts, err
	}

	return arts, nil
}
