package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represnts a parsed config.yaml file.
//
// Fields:
//   - base-img-path: Local path or docker:// URL for the base installation image
//   - temp-dir: Local directory used to extract and modify the image. If unprovided,
//     one will be created inside of the install-dir.
//   - install-dir: Local directory where the final container image and generated
//     symlinks will be stored.
//   - wrapper-script: Local path to the wrapper script that generated symlinks
//     will point to.
//   - apt-src: Local directory, HTTP URL, or s3:// URL pointing to the APT
//     repository used to install packages into the image.
//   - module-path: Local directory where generated module files will be placed.
//   - artefact-store: Local path or s3:// URL used to store generated artefacts.
//   - db-conn: Database connection string, either a local SQLite path or a
//     mysql:// style URI for a remote database.
//   - listen-addr: Address where the web server will listen.
type Config struct {
	BaseImgPath   string `yaml:"base-img-path"`
	TempDir       string `yaml:"temp-dir"`
	InstallDir    string `yaml:"install-dir"`
	WrapperScript string `yaml:"wrapper-script"`
	AptSrc        string `yaml:"apt-src"`
	AptIndexSrc   string `yaml:"apt-index-src"`

	ModulePath    string `yaml:"module-path"`
	ArtefactStore string `yaml:"artefact-store"`
	DBConn        string `yaml:"db-conn"`
	Driver        string `yaml:"db-driver"`
	ListenAddr    string `yaml:"listen-addr"`
}

// Load will load the config at the provided path.
func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

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
