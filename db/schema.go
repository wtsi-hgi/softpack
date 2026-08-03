package db

type Environment struct {
	ID          uint      `gorm:"primaryKey"                    json:"-"`
	Name        string    `gorm:"not null;uniqueIndex:capybara" json:"name"`
	Path        string    `gorm:"not null;uniqueIndex:capybara" json:"path"`
	Version     int       `gorm:"not null;uniqueIndex:capybara" json:"version"`
	Description string    `gorm:"not null"                      json:"description"`
	Created     int       `gorm:"not null"                      json:"created"`
	Hidden      bool      `gorm:"not null"                      json:"hidden"`
	Tags        []Tag     `gorm:"many2many:environment_tags"`
	Packages    []Package `gorm:"not null;serializer:json"      json:"packages"`
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
	Name      string `gorm:"not null"  json:"name"`
	Version   string `gorm:"not null"  json:"version"`
	URL       string `gorm:"not null"  json:"url"`
	Details   string `gorm:"not null"  json:"details"`
	Requester string `json:"requester"`
}

// path/name-version

type Tag struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex;not null" json:"tag"`
}
