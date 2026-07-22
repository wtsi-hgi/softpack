package db

type Environment struct {
	Name        string   `gorm:"not null;uniqueIndex:capybara"`
	Path        string   `gorm:"not null;uniqueIndex:capybara"`
	Version     int      `gorm:"not null;uniqueIndex:capybara"`
	Description string   `gorm:"not null"`
	Created     int      `gorm:"not null"`
	Hidden      bool     `gorm:"not null"`
	Tags        []string `gorm:"not null;serializer:json"`
	Packages    []string `gorm:"not null;serializer:json"`
}

// TODO: is requester required?
type RecipeRequest struct {
	// ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"not null"`
	Version   string `gorm:"not null"`
	URL       string `gorm:"not null"`
	Details   string `gorm:"not null"`
	Requester string
}

// path/name-version

// TODO: Add tags table?
