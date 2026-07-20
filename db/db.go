package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ErrMissingField = errors.New("one or more required fields missing")
)

type DB struct {
	*gorm.DB
}

func Connect(driver, connection string) (*DB, error) {
	var (
		db  *gorm.DB
		err error
	)

	switch driver {
	case "sqlite", "sqlite3":
		db, err = gorm.Open(sqlite.Open(connection), &gorm.Config{})
	case "mysql":
		db, err = gorm.Open(mysql.Open(connection), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Environment{}, &RecipeRequest{})
	if err != nil {
		return nil, err
	}

	return &DB{db}, err
}

func (e *Environment) BeforeCreate(tx *gorm.DB) error {
	if e.Name == "" || e.Path == "" || e.Version == 0 || e.Created == 0 {
		return ErrMissingField
	}

	return nil
}

func (e *Environment) ToIndex() EnvironmentIndex {
	return EnvironmentIndex{
		Name:    e.Name,
		Path:    e.Path,
		Version: e.Version,
	}
}

// func (db *DB) dropTables(tables ...interface{}) error {
// 	return db.Migrator().DropTable(tables...)
// }

// TODO: Should I keep this?
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

func (db *DB) CreateEnvironment(ctx context.Context, env Environment) error {
	return db.WithContext(ctx).Create(&env).Error
}

// func (db *DB) UpdateEnvironment(ctx context.Context, index EnvironmentIndex, updates map[string]interface{}) error {
// 	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
// 		Name:    index.Name,
// 		Path:    index.Path,
// 		Version: index.Version,
// 	}).Updates(updates).Error
// }

func (db *DB) UpdateEnvironment(ctx context.Context, env Environment) error {
	return db.WithContext(ctx).Model(&Environment{}).Where(&Environment{
		Name:    env.Name,
		Path:    env.Path,
		Version: env.Version,
	}).Updates(&env).Error
}

// GetEnvironments retrieves environments from the database.
// Provide no additional arguments to retrieve all environments.
// Provide one or more EnvironmentIndex items to retrieve specific environments.
func (db *DB) GetEnvironments(ctx context.Context, indexes ...EnvironmentIndex) ([]Environment, error) {
	var envs []Environment
	query := db.WithContext(ctx)

	if len(indexes) == 0 {
		if err := query.Find(&envs).Error; err != nil {
			return nil, err
		}

		return envs, nil
	}

	for _, index := range indexes {
		var env Environment

		r := query.Where(&Environment{
			Name:    index.Name,
			Path:    index.Path,
			Version: index.Version,
		}).First(&env)

		if r.Error != nil {
			return nil, r.Error
		}

		envs = append(envs, env)
	}

	return envs, nil
}

func (db *DB) DeleteEnvironment(ctx context.Context, index EnvironmentIndex) error {
	return db.WithContext(ctx).Where(&Environment{
		Name:    index.Name,
		Path:    index.Path,
		Version: index.Version,
	}).Delete(&Environment{}).Error
}
