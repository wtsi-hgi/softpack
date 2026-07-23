package db

type Environment struct {
	Name        string   `gorm:"not null;uniqueIndex:capybara"`
	Path        string   `gorm:"not null;uniqueIndex:capybara"`
	Version     int      `gorm:"not null;uniqueIndex:capybara"`
	Description string   `gorm:"not null"`
	Created     int      `gorm:"not null"`
	Hidden      bool     `gorm:"not null"`
	Tags        []string `gorm:"not null;serializer:json"`
	Packages    []string `gorm:"not null;serializer:json"` // type Package = { name: string; version?: string | null | undefined; }
	// readme string
	// envtype type EnvironmentType = "softpack" | "module";
	// username? string
	// failure_reason? string
	// interpreters type Interpreters = { r?: string | undefined; python?: string | undefined; }
}

// TODO: is requester required?
type RecipeRequest struct {
	// ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"not null"`
	Version   string `json:"version" gorm:"not null"`
	URL       string `json:"url" gorm:"not null"`
	Details   string `json:"details" gorm:"not null"`
	Requester string `json:"requester"`
}

// path/name-version

// TODO: Add tags table?
