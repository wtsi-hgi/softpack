package db

type Environment struct {
	Name        string    `json:"name" gorm:"not null;uniqueIndex:capybara"`
	Path        string    `json:"path" gorm:"not null;uniqueIndex:capybara"`
	Version     int       `json:"version" gorm:"not null;uniqueIndex:capybara"`
	Description string    `json:"description" gorm:"not null"`
	Created     int       `json:"created" gorm:"not null"`
	Hidden      bool      `json:"hidden" gorm:"not null"`
	Tags        []string  `json:"tags" gorm:"not null;serializer:json"`
	Packages    []Package `json:"packages" gorm:"not null;serializer:json"`
	// readme string
	// envtype type EnvironmentType = "softpack" | "module";
	// username? string
	// failure_reason? string
	// interpreters type Interpreters = { r?: string | undefined; python?: string | undefined; }
}

// TODO: Should I use this Package definition or the one from apt?
type Package struct {
	Name        string   `json:"name"`
	Description string   `json:"-"`
	Versions    []string `json:"version"`
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
