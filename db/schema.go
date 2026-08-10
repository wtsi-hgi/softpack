package db

type Status int

const (
	Waiting Status = iota
	Building
	Concretised
	Failed
)

type Environment struct {
	ID            uint      `gorm:"primaryKey"                    json:"-"`
	Name          string    `gorm:"not null;uniqueIndex:capybara" json:"name"`
	Path          string    `gorm:"not null;uniqueIndex:capybara" json:"path"`
	Version       int       `gorm:"not null;uniqueIndex:capybara" json:"version"`
	Description   string    `gorm:"not null"                      json:"description"`
	Created       int       `gorm:"not null"                      json:"created"`
	Hidden        bool      `gorm:"not null"                      json:"hidden"`
	Tags          []Tag     `gorm:"many2many:environment_tags"`
	Packages      []Package `gorm:"not null;serializer:json"      json:"packages"`
	Status        Status
	BuildStart    int64
	BuildEnd      int64
	FailureReason string
	// envtype type EnvironmentType = "softpack" | "module";

	// username? string
}

type Package struct {
	Name        string `json:"name"`
	Description string `json:"-"`
	Version     string `json:"version"`
	Interpreter bool   `json:"interpreter"`
}

type RecipeRequest struct {
	Name      string `gorm:"not null"  json:"name"`
	Version   string `gorm:"not null"  json:"version"`
	URL       string `gorm:"not null"  json:"url"`
	Details   string `gorm:"not null"  json:"details"`
	Requester string `json:"requester"`
}

type Tag struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex;not null" json:"tag"`
}
