package db

type Environment struct {
	Name        string `gorm:"not null;uniqueIndex:capybara"`
	Path        string `gorm:"not null;uniqueIndex:capybara"`
	Version     int    `gorm:"not null;uniqueIndex:capybara"`
	Description string
	Created     int `gorm:"not null"`
	Hidden      bool
	Tags        []string `gorm:"serializer:json"`
	Packages    []string `gorm:"serializer:json"`
}

type EnvironmentIndex struct {
	Name, Path string
	Version    int
}

type RecipeRequest struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	Name      string
	Version   string
	URL       string
	Details   string
	Requester string
}

// path/name-version
