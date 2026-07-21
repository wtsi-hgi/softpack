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

type EnvironmentIndex struct {
	Name, Path string
	Version    int
}

type UpdateByIndex struct {
	EnvironmentIndex
	Value string
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
