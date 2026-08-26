package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/softpack/db"
	"gopkg.in/yaml.v3"
)

var (
	ErrArgs      = errors.New("invalid args number")
	ErrNoVersion = errors.New("no version found in name (envName-[version])")
)

var importCmd = &cobra.Command{
	Use:   "import <artifactRootPath>",
	Short: "Import softpack git repo artefacts to database",
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) != 1 {
			return ErrArgs
		}

		root := args[0]

		matches, err := filepath.Glob(root + "/environments/*/*/*")
		if err != nil {
			return err
		}

		for _, path := range matches[:1] {
			env, err := createEnvFromDir(root, path)
			if err != nil {
				return err
			}

			fmt.Printf("Environment: %+v\n", env)
		}

		return nil
	},
}

func createEnvFromDir(root, path string) (*db.Environment, error) {
	_, files, err := readDir(path)
	if err != nil {
		return nil, err
	}

	env, err := getEnvIndex(root, path)
	if err != nil {
		return nil, err
	}

	if slices.Contains(files, ".built_by_softpack") {
		env.Type = db.Softpack
	} else {
		env.Type = db.Module
	}

	// TODO: do i need this check since the next func will error anyway if this is the case?
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

func getEnvIndex(root, path string) (*db.Environment, error) {
	pathItems := strings.Split(path, "/")
	name := pathItems[len(pathItems)-1]

	version, err := getVersionFromName(name)
	if err != nil {
		return nil, err
	}

	return &db.Environment{
		Name:    name,
		Path:    strings.TrimPrefix(path, root),
		Version: version,
	}, nil
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

	// TODO: Packages

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

	// TODO: Tags

	env.Created = contents.Created
	env.Hidden = contents.Hidden
	env.FailureReason = contents.FailureReason

	return nil
}

func getVersionFromName(name string) (int, error) {
	i := strings.LastIndex(name, "-")
	if i == -1 {
		return -1, ErrNoVersion
	}

	n, err := strconv.Atoi(name[i+1:])
	if err != nil {
		return -1, err
	}

	return n, nil
}

func init() {
	RootCmd.AddCommand(importCmd)
}
