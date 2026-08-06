package db

import (
	"context"
	"errors"

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
type UpdateValue[T any] struct { // TODO: I dont like this, unnecessary duplication with updateenv //nolint:godox
	EnvironmentIndex
	Value T
}

// UpdateEnv allows multiple fields of an environment record, identified by EnvironmentIndex,
// to be updated.
type UpdateEnv struct {
	EnvironmentIndex
	Description *string
	Hidden      *bool
	Tags        *[]Tag
	Status      *int
}

func (e *Environment) BeforeCreate(_ *gorm.DB) error {
	if e.Name == "" || e.Path == "" || e.Version == 0 || e.Created == 0 {
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

func (u *UpdateValue[T]) ToIndex() EnvironmentIndex {
	return EnvironmentIndex{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}
}

// CreateEnvironments will add the given Environments to the database.
func (db *DB) CreateEnvironments(ctx context.Context, envs []Environment) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, env := range envs {
			if err := gorm.G[Environment](tx).Create(ctx, &env); err != nil {
				return err
			}
		}

		return nil
	})
}

// CreateEnvironment will add the given Environment to the database.
func (db *DB) CreateEnvironment(ctx context.Context, env Environment) error {
	return db.WithContext(ctx).Create(&env).Error
}

// UpdateEnvironment will update the environment record specified by the EnvironmentIndex
// to match the non-nil UpdateEnv fields.
// Providing 'Tags' here will result in all previous tags being removed and subsequently
// replaced with the provided ones, to add/delete tags, consider using <Add/Delete>EnvironmentTag.
// func (db *DB) UpdateEnvironment(ctx context.Context, u UpdateEnv) error {
// 	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// 		updates := collectUpdates(u)

// 		e, err := preLoadEnv(tx, u)
// 		if err != nil {
// 			return err
// 		}

// 		if len(updates) > 0 {
// 			if err := tx.Model(e).Where(&Environment{
// 				Name:    u.Name,
// 				Path:    u.Path,
// 				Version: u.Version,
// 			}).Updates(updates).Error; err != nil {
// 				return err
// 			}
// 		}

// 		if u.Tags != nil {
// 			if err := replaceTags(tx, u, *e); err != nil {
// 				return err
// 			}
// 		}

// 		return nil
// 	})
// }

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

// func(db *DB)

// func collectUpdates(u UpdateEnv) map[string]any {
// 	updates := map[string]any{}

// 	if u.Description != nil {
// 		updates["Description"] = *u.Description
// 	}

// 	if u.Hidden != nil {
// 		updates["Hidden"] = *u.Hidden
// 	}

// 	if u.Status != nil {
// 		updates["Status"] = *u.Status
// 	}

// 	return updates
// }

// func preLoadEnv(tx *gorm.DB, u UpdateEnv) (*Environment, error) {
// 	var e Environment

// 	if err := tx.Where(&Environment{
// 		Name:    u.Name,
// 		Path:    u.Path,
// 		Version: u.Version,
// 	}).First(&e).Error; err != nil {
// 		return nil, err
// 	}

// 	return &e, nil
// }

// func replaceTags(tx *gorm.DB, u UpdateEnv, e Environment) error {
// 	tags := *u.Tags

// 	for i := range tags {
// 		if err := tx.FirstOrCreate(&tags[i], Tag{
// 			Name: tags[i].Name,
// 		}).Error; err != nil {
// 			return err
// 		}
// 	}

// 	if err := tx.Model(&e).Association("Tags").Replace(tags); err != nil {
// 		return err
// 	}

// 	return nil
// }

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
	var env Environment
	if err := db.WithContext(ctx).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).First(&env).Error; err != nil {
		return err
	}

	var tag Tag
	if err := db.WithContext(ctx).Where(&Tag{
		Name: u.Value.Name,
	}).First(&tag).Error; err != nil {
		return err
	}

	return db.WithContext(ctx).
		Model(&env).
		Association("Tags").
		Delete(&tag)
}

func (db *DB) GetTags(ctx context.Context) ([]Tag, error) {
	var tags []Tag

	if err := db.WithContext(ctx).Find(&tags).Error; err != nil {
		return nil, err
	}

	return tags, nil
}

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

func (db *DB) UpdateEnvPackage(ctx context.Context, u UpdateValue[FulfilRequestBody]) error {
	var env Environment

	if err := db.WithContext(ctx).Where(&Environment{
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

	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
		Name:    u.Name,
		Path:    u.Path,
		Version: u.Version,
	}).Update("Packages", env.Packages).Error
}
