package db

import (
	"errors"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ErrMissingField      = errors.New("one or more required fields missing")
	ErrUnsupportedDriver = errors.New("unsupported driver")
)

type DB struct {
	*gorm.DB
}

// Connect connects to a database given a driver and connection string.
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
		return nil, ErrUnsupportedDriver
	}

	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Environment{}, &RecipeRequest{}, &Tag{})
	if err != nil {
		return nil, err
	}

	return &DB{db}, err
}
