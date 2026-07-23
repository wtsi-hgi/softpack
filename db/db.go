package db

import (
	"errors"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TODO: Should i split this into seperate files for env related stuff and recipe related stuff and db setup stuff?

var ErrMissingField = errors.New("one or more required fields missing")

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

// func (db *DB) dropTables(tables ...interface{}) error {
// 	return db.Migrator().DropTable(tables...)
// }
