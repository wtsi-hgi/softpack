package install

import (
	"os"
	"path/filepath"

	"github.com/wtsi-hgi/softpack/artefacts"
	"github.com/wtsi-hgi/softpack/build"
	"github.com/wtsi-hgi/softpack/module"
)

func Install(baseImage, moduleBase, tempDir, installBase, wrapperScript, artefactBase, envPath, envName, envVer, aptSrc, description string, pkgs []build.Package) (*build.Artefacts, error) {
	installPath := filepath.Join(installBase, envPath, envName, envVer+"-scripts")

	if err := os.MkdirAll(installPath, 0755); err != nil {
		return nil, err
	}

	arts, err := build.Build(baseImage, tempDir, installPath, wrapperScript, aptSrc, pkgs)
	if err != nil {
		return nil, err
	}

	if err = module.Install(moduleBase, installBase, envPath, envName, envVer, description, arts.Exes, arts.Packages); err != nil {
		return nil, err
	}

	if err := artefacts.Store(artefactBase, filepath.Join(envPath, envName, envVer), map[string]artefacts.Opener{
		"module":          artefacts.File(module.ModuleFile(moduleBase, envPath, envName, envVer)),
		"singularity.sif": artefacts.File(build.SingularityPath(installPath)),
		"build.log":       artefacts.Data(arts.Log),
	}); err != nil {
		return nil, err
	}

	return arts, nil
}
