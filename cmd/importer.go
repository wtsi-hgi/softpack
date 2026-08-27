package cmd

import (
	"bytes"
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
)

var importCmd = &cobra.Command{
	Use:   "import <artifactRootPath> [databasePath]",
	Short: "Import softpack git repo artefacts to database",
	Long: `Import softpack git repo artefacts to database.
	
Provide the path to the root of the artefacts repo, and optionally a path to a
database to add the environments to. If none is specified, one called 'import.db'
will be created.
`,
	Args: cobra.RangeArgs(1, 2), //nolint:mnd
	RunE: func(_ *cobra.Command, args []string) error {
		root := args[0]

		dbPath := "import.db"
		if len(args) == 2 { //nolint:mnd
			dbPath = args[1]
		}

		matches, err := filepath.Glob(root + "/environments/users/*/*")
		if err != nil {
			return err
		}

		envs, err := generateEnvs(matches, root)
		if err != nil {
			return err
		}

		db, err := db.Connect("sqlite3", dbPath)
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

func createEnvFromDir(root, path string) (*db.Environment, error) { //nolint:funlen
	_, files, err := readDir(path)
	if err != nil {
		return nil, err
	}

	env := getEnvIndex(root, path)

	if slices.Contains(files, ".built_by_softpack") {
		env.Type = db.Softpack
	} else {
		env.Type = db.Module
	}

	populateRequester(env, path)

	if slices.Contains(files, "softpack.yml") {
		if err := populateEnvFromSoftpackYML(env, path); err != nil {
			return nil, err
		}
	}

	if slices.Contains(files, "meta.yml") {
		if err := populateEnvFromMetaYML(env, path); err != nil {
			return nil, err
		}
	}

	if slices.Contains(files, "README.md") {
		if err := populateEnvReadMe(env, path); err != nil {
			return nil, err
		}
	}

	if slices.Contains(files, "spack.lock") {
		if err := collectInterpreters(env, path); err != nil {
			return nil, err
		}
	}

	populateStatus(env, files)

	return env, nil
}

func readDir(path string) ([]os.DirEntry, []string, error) {
	dirEntry, err := os.ReadDir(path)
	if err != nil {
		return nil, nil, err
	}

	var files []string

	for _, item := range dirEntry {
		info, err := item.Info()
		if err != nil {
			return nil, nil, err
		}

		files = append(files, info.Name())
	}

	return dirEntry, files, nil
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
	f, err := os.ReadFile(filepath.Join(path, "spack.lock"))
	if err != nil {
		return err
	}

	var contents SpackLock

	if err := json.NewDecoder(bytes.NewReader(f)).Decode(&contents); err != nil {
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

	env.Status = db.Failed // TODO: I dont like assuming this
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

	f, err := os.ReadFile(filepath.Join(path, "softpack.yml"))
	if err != nil {
		return err
	}

	if err := yaml.NewDecoder(bytes.NewReader(f)).Decode(&contents); err != nil {
		return err
	}

	env.Description = contents.Description

	var pkgs []db.Package

	for _, pkg := range contents.Packages {
		parts := strings.Split(pkg, "@")

		version := ""
		if len(parts) == 2 { //nolint:mnd
			version = parts[1]
		}

		pkgs = append(pkgs, db.Package{
			Name:    parts[0],
			Version: version,
			// TODO: Interpreter??
		})
	}

	env.Packages = pkgs

	return nil
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

	f, err := os.ReadFile(filepath.Join(path, "meta.yml"))
	if err != nil {
		return err
	}

	if err := yaml.NewDecoder(bytes.NewReader(f)).Decode(&contents); err != nil {
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
}
