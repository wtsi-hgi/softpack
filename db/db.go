package db

import (
	"errors"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	}

	switch driver {
	case "sqlite", "sqlite3":
		db, err = gorm.Open(sqlite.Open(connection), config)
	case "mysql":
		db, err = gorm.Open(mysql.Open(connection), config)
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
