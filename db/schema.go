package db

type Environment struct {
	ID          uint      `json:"-" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex:capybara"`
	Path        string    `json:"path" gorm:"not null;uniqueIndex:capybara"`
	Version     int       `json:"version" gorm:"not null;uniqueIndex:capybara"`
	Description string    `json:"description" gorm:"not null"`
	Created     int       `json:"created" gorm:"not null"`
	Hidden      bool      `json:"hidden" gorm:"not null"`
	Tags        []Tag     `gorm:"many2many:environment_tags"`
	Packages    []Package `json:"packages" gorm:"not null;serializer:json"`
	// Tags     []string  `json:"tags" gorm:"not null;serializer:json"`
	// status
	// readme string
	// envtype type EnvironmentType = "softpack" | "module";
	// username? string
	// failure_reason? string
	// interpreters type Interpreters = { r?: string | undefined; python?: string | undefined; }
}

type Package struct {
	Name        string `json:"name"`
	Description string `json:"-"`
	Version     string `json:"version"`
}

type RecipeRequest struct {
	Name      string `json:"name"    gorm:"not null"`
	Version   string `json:"version" gorm:"not null"`
	URL       string `json:"url"     gorm:"not null"`
	Details   string `json:"details" gorm:"not null"`
	Requester string `json:"requester"`
}

// path/name-version

type Tag struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `json:"tag" gorm:"uniqueIndex;not null"`
}
