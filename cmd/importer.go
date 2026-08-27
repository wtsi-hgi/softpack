package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/softpack/db"
	"gopkg.in/yaml.v3"
)

var (
	ErrArgs           = errors.New("invalid args number")
	ErrInvalidVersion = errors.New("invalid version")

	ArtifactRootPath string
	DatabasePath     string
	Driver           string
)

var importCmd = &cobra.Command{
	Use:   "import <artifactRootPath> [databasePath] [driver]",
	Short: "Import softpack git repo artefacts to database",
	Long: `Import softpack git repo artefacts to database.
	
artifactRootPath: Path to the root of the artefacts repo.
[Optional] databasePath: Path to a database to add the environments to. If 
none is specified, one called 'import.db' will be created.
[Optional] driver: Driver to connect to the database with. Defaults to 'sqlite3'.
`,
	RunE: func(_ *cobra.Command, _ []string) error {
		dbPath := "import.db"
		if DatabasePath != "" {
			dbPath = DatabasePath
		}

		driver := "sqlite3"
		if Driver != "" {
			driver = Driver
		}

		matches, err := filepath.Glob(ArtifactRootPath + "/environments/users/*/*")
		if err != nil {
			return err
		}

		envs, err := generateEnvs(matches, ArtifactRootPath)
		if err != nil {
			return err
		}

		db, err := db.Connect(driver, dbPath)
		if err != nil {
			return err
		}

		if err := db.CreateEnvironments(context.Background(), envs); err != nil {
			return err
		}

		return nil
	},
}

func generateEnvs(matches []string, root string) ([]db.Environment, error) {
	var envs []db.Environment

	for _, path := range matches {
		env, err := createEnvFromDir(root, path)
		if err != nil {
			return nil, err
		}

		envs = append(envs, *env)
	}

	return envs, nil
}

func createEnvFromDir(root, path string) (*db.Environment, error) {
	files, err := readDir(path)
	if err != nil {
		return nil, err
	}

	env := getEnvIndex(root, path)
	env.Type = db.Module

	if slices.Contains(files, ".built_by_softpack") {
		env.Type = db.Softpack
	}

	populateRequester(env, path)

	for file, handler := range map[string]func(*db.Environment, string) error{
		"softpack.yml": populateEnvFromSoftpackYML,
		"meta.yml":     populateEnvFromMetaYML,
		"README.md":    populateEnvReadMe,
		"spack.lock":   collectInterpreters,
	} {
		if slices.Contains(files, file) {
			if err := handler(env, path); err != nil {
				return nil, err
			}
		}
	}

	populateStatus(env, files)

	return env, nil
}

func readDir(path string) ([]string, error) {
	dirEntry, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []string

	for _, item := range dirEntry {
		info, err := item.Info()
		if err != nil {
			return nil, err
		}

		files = append(files, info.Name())
	}

	return files, nil
}

func getEnvIndex(root, path string) *db.Environment {
	pathItems := strings.Split(path, "/")
	name := pathItems[len(pathItems)-1]

	version := getVersionFromName(name)

	return &db.Environment{
		Name:    name,
		Path:    strings.TrimSuffix(strings.TrimPrefix(path, root+"environments/"), name),
		Version: version,
	}
}

type SpackLock struct {
	Specs map[string]dependency `json:"concrete_specs"`
}

type dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func collectInterpreters(env *db.Environment, path string) error {
	f, err := os.Open(filepath.Join(path, "spack.lock"))
	if err != nil {
		return err
	}
	defer f.Close()

	var contents SpackLock

	if err := json.NewDecoder(f).Decode(&contents); err != nil {
		return err
	}

	for _, dep := range contents.Specs {
		if dep.Name == "python" || dep.Name == "r" {
			env.Packages = append(env.Packages, db.Package{
				Name:        dep.Name,
				Version:     dep.Version,
				Interpreter: true,
			})
		}
	}

	return nil
}

func populateStatus(env *db.Environment, files []string) {
	if slices.Contains(files, "module") {
		env.Status = db.Concretised

		return
	}

	for _, pkg := range env.Packages {
		if strings.HasPrefix(pkg.Name, "*") {
			env.Status = db.Waiting

			return
		}
	}

	env.Status = db.Failed
}

func populateRequester(env *db.Environment, path string) {
	if strings.Contains(path, "users") {
		parts := strings.Split(path, "/")
		idx := slices.Index(parts, "users")
		env.Requester = parts[idx+1]
	}
}

func populateEnvReadMe(env *db.Environment, path string) error {
	f, err := os.ReadFile(filepath.Join(path, "README.md"))
	if err != nil {
		return err
	}

	env.Readme = string(f)

	return nil
}

type softpackYML struct {
	Description string   `yaml:"description"`
	Packages    []string `yaml:"packages"`
}

// populateEnvFromSoftpackYML will populate the description and Package fields.
func populateEnvFromSoftpackYML(env *db.Environment, path string) error {
	var contents softpackYML

	f, err := os.Open(filepath.Join(path, "softpack.yml"))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&contents); err != nil {
		return err
	}

	env.Description = contents.Description
	env.Packages = parsePackages(contents.Packages)

	return nil
}

func parsePackages(packages []string) []db.Package {
	pkgs := make([]db.Package, 0, len(packages))

	for _, pkg := range packages {
		name, version, _ := strings.Cut(pkg, "@")

		pkgs = append(pkgs, db.Package{
			Name:    name,
			Version: version,
		})
	}

	return pkgs
}

type metaYML struct {
	Tags          []string `yaml:"tags"`
	Created       int      `yaml:"created"`
	Hidden        bool     `yaml:"hidden"`
	FailureReason string   `yaml:"failure_reason"`
}

// populateEnvFromMetaYML will populate the Created, Hidden, FailureReason and
// Tags fields.
func populateEnvFromMetaYML(env *db.Environment, path string) error {
	var contents metaYML

	f, err := os.Open(filepath.Join(path, "meta.yml"))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&contents); err != nil {
		return err
	}

	var tags []db.Tag

	for _, n := range contents.Tags {
		tags = append(tags, db.Tag{Name: n})
	}

	env.Tags = tags
	env.Created = contents.Created
	env.Hidden = contents.Hidden
	env.FailureReason = contents.FailureReason

	return nil
}

func getVersionFromName(name string) string {
	i := strings.LastIndex(name, "-")
	if i == -1 {
		return "1"
	}

	return name[i+1:]
}

func init() {
	RootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&ArtifactRootPath, "artifactRootPath", "a", "", "Path to the root of the artefacts repo.")
	importCmd.Flags().StringVarP(&DatabasePath, "databasePath", "t", "", "Path to a database to add the environments to.")
	importCmd.Flags().StringVarP(&Driver, "driver", "d", "", "Database driver.")

	importCmd.MarkFlagRequired("artifactRootPath") //nolint:errcheck
}
