package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// BaseImage - local path, or docker:// url for the base install image (currently, an Ubuntu Resolute image).
// TempDir - local directory where the image will be extracted and modified; this can be empty and it will create a temp directory in the final install directory.
// InstallDir - local directory where the final container and symlinks will be put.
// WrapperScript - local path to wrapper script that the symlinks will point to.
// AptSrc - local directory, http:// URL, or s3:// URL pointing to the APT repo that will be used to install.

// In addition, we will also need:

// ModulePath - local path to place the generated module files.
// ArtefactStore - local path or s3:// URL used to store generated artefacts.
// DBConn - either pointing to a local sqlite location, or a mysql:// style URI for a remote DB.
// ListenAddr - address that the webserver will listen on; can default to something like ":8080" if not specified.
type Config struct {
	BaseImgPath   string `yaml:"base-img-path"`
	TempDir       string `yaml:"temp-dir"`
	InstallDir    string `yaml:"install-dir"`
	WrapperScript string `yaml:"wrapper-script"`
	AptSrc        string `yaml:"apt-src"`

	ModulePath    string `yaml:"module-path"`
	ArtefactStore string `yaml:"artefact-store"`
	DBConn        string `yaml:"db-conn"`
	ListenAddr    string `yaml:"listen-addr"`
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConf()

	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.TempDir == "" {
		dir, err := os.MkdirTemp(cfg.InstallDir, "tmp")
		if err != nil {
			return nil, err
		}

		cfg.TempDir = dir
	}

	return cfg, nil
}

func DefaultConf() *Config {
	return &Config{
		ListenAddr: "8080",
	}
}
