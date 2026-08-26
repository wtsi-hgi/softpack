package db

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNoRowsAffected = errors.New("no rows affected by query")

// EnvironmentIndex represents a multiindex used to uniquely identify a record
// in the Environment table.
type EnvironmentIndex struct {
	Name, Path string
	Version    int
}

// UpdateValue allows one specific field of an environment record, identified by
// EnvironmentIndex, to be updated.
type UpdateValue[T any] struct {
	EnvironmentIndex
	Value T
}

func (e *Environment) BeforeCreate(_ *gorm.DB) error {
	if e.Name == "" || e.Path == "" || e.Version == 0 {
		return ErrMissingField
	}

	return nil
}

// ToIndex returns an EnvironmentIndex that will uniquely identify the environment
// it is called upon.
func (e *Environment) ToIndex() EnvironmentIndex {
	return EnvironmentIndex{
		Name:    e.Name,
		Path:    e.Path,
		Version: e.Version,
	}
}

// CreateEnvironments will add the given Environments to the database.
func (db *DB) CreateEnvironments(ctx context.Context, envs []Environment) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, env := range envs {
			version, err := getNextEnvVersion(ctx, tx, env.Name, env.Path)
			if err != nil {
				return err
			}

			env.Version = version
			if err := gorm.G[Environment](tx).Create(ctx, &env); err != nil {
				return err
			}
		}

		return nil
	})
}

// CreateEnvironment will add the given Environment to the database.
func (db *DB) CreateEnvironment(ctx context.Context, env *Environment) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := getNextEnvVersion(ctx, tx, env.Name, env.Path)
		if err != nil {
			return err
		}

		env.Created = int(time.Now().Unix())
		env.Version = version
		env.Type = Softpack

		if err := tx.WithContext(ctx).Create(env).Error; err != nil {
			return err
		}

		return nil
	})
}

func (db *DB) UpdateHidden(ctx context.Context, u UpdateValue[bool]) error {
	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).Update("Hidden", u.Value).Error
}

func (db *DB) UpdateStatus(ctx context.Context, u UpdateValue[Status]) error {
	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).Update("Status", u.Value).Error
}

func (db *DB) UpdateStatusWithFailureReason(ctx context.Context, u UpdateValue[string]) error {
	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).Updates(map[string]any{
		"Status":        Failed,
		"FailureReason": u.Value,
	}).Error
}

// GetEnvironments retrieves all environments from the database.
func (db *DB) GetEnvironments(ctx context.Context) ([]Environment, error) {
	var envs []Environment

	if err := db.WithContext(ctx).Preload(clause.Associations).Find(&envs).Error; err != nil {
		return nil, err
	}

	return envs, nil
}

func (db *DB) DeleteEnvironment(ctx context.Context, index EnvironmentIndex) error {
	result := db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Where(&Environment{
		Name:    index.Name,
		Path:    index.Path,
		Version: index.Version,
	}).Delete(&Environment{})

	err := result.Error
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

func (db *DB) AddEnvironmentTag(ctx context.Context, u UpdateValue[Tag]) error {
	var env Environment

	if err := db.WithContext(ctx).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).First(&env).Error; err != nil {
		return err
	}

	return db.WithContext(ctx).Model(&env).Association("Tags").Append(u.Value)
}

func (db *DB) DeleteEnvironmentTag(ctx context.Context, u UpdateValue[Tag]) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var env Environment
		if err := tx.WithContext(ctx).Where(&Environment{
			Name:    u.Name,
			Path:    u.Path,
			Version: u.Version,
		}).First(&env).Error; err != nil {
			return err
		}

		var tag Tag
		if err := tx.WithContext(ctx).Where(&Tag{
			Name: u.Value.Name,
		}).First(&tag).Error; err != nil {
			return err
		}

		if err := tx.WithContext(ctx).Model(&env).Association("Tags").Delete(&tag); err != nil {
			return err
		}

		return tx.WithContext(ctx).Model(&Tag{}).Where(Tag{ID: tag.ID}).Delete(&Tag{}).Error
	})
}

func (db *DB) GetTags(ctx context.Context) ([]Tag, error) {
	var tags []Tag

	if err := db.WithContext(ctx).Find(&tags).Error; err != nil {
		return nil, err
	}

	return tags, nil
}

// Concretise will replace the given environment's packages with those provided.
func (db *DB) Concretise(env Environment, pkgs []Package) error {
	return db.Model(&env).Where(&Environment{
		Name:    env.Name,
		Path:    env.Path,
		Version: env.Version,
	}).Updates(&Environment{
		Packages: pkgs,
	}).Error
}

type FulfilRequestBody struct {
	RecipeRequest
	CanonicalName    string
	CanonicalVersion string
}

// FulfilEnvPackage will update an environment's package information to match a newly
// fulfilled request for a package that it relies on.
func (db *DB) FulfilEnvPackage(ctx context.Context, u UpdateValue[FulfilRequestBody]) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var env Environment

		if err := tx.WithContext(ctx).Where(&Environment{
			Name:    u.Name,
			Path:    u.Path,
			Version: u.Version,
		}).First(&env).Error; err != nil {
			return err
		}

		for n, pkg := range env.Packages {
			if CheckPkgEqual(pkg, u.Value.RecipeRequest) {
				env.Packages[n].Name = u.Value.CanonicalName
				env.Packages[n].Version = u.Value.CanonicalVersion

				break
			}
		}

		return tx.WithContext(ctx).Save(&env).Error
	})
}

func getNextEnvVersion(ctx context.Context, tx *gorm.DB, name, path string) (int, error) {
	var envs []Environment

	if err := tx.WithContext(ctx).Preload(clause.Associations).Where(&Environment{
		Name: name,
		Path: path,
	}).Find(&envs).Error; err != nil {
		return -1, err
	}

	if len(envs) == 0 {
		return 1, nil
	}

	highest := -1

	for _, env := range envs {
		if env.Version > highest {
			highest = env.Version
		}
	}

	return highest + 1, nil
}

func (db *DB) SetEnvBuildTime(ctx context.Context, u UpdateValue[int64], start bool) error {
	var updateField string

	if start {
		updateField = "BuildStart"
	} else {
		updateField = "BuildEnd"
	}

	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).Update(updateField, u.Value).Error
}

func (db *DB) GetBuildingEnvs(ctx context.Context) (envs []Environment, err error) {
	if err := db.WithContext(ctx).Model(&Environment{}).Where(Environment{
		Status: Building,
	}).Find(&envs).Error; err != nil {
		slog.Error("Failure retrieving building envs", "err", err)

		return []Environment{}, err
	}

	return envs, nil
}
