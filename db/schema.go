package db

type Status int

const (
	Waiting Status = iota
	Building
	Concretised
	Failed
)

type EnvType int

const (
	Softpack EnvType = iota
	Module
)

type Environment struct {
	ID            uint      `gorm:"primaryKey"                    json:"-"`
	Name          string    `gorm:"not null;uniqueIndex:envIndex" json:"name"`
	Path          string    `gorm:"not null;uniqueIndex:envIndex" json:"path"`
	Version       int       `gorm:"not null;uniqueIndex:envIndex" json:"version"`
	Description   string    `gorm:"not null"                      json:"description"`
	Created       int       `gorm:"not null"                      json:"created"`
	Hidden        bool      `gorm:"not null"                      json:"hidden"`
	Tags          []Tag     `gorm:"many2many:environment_tags"    json:"tags"`
	Packages      []Package `gorm:"not null;serializer:json"      json:"packages"`
	Type          EnvType   `json:"type"`
	Status        Status    `json:"status"`
	BuildStart    int64     `json:"buildstart"`
	BuildEnd      int64     `json:"buildend"`
	FailureReason string    `json:"failurereason"`
	Requester     string    `json:"requester"`
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
	Name string `gorm:"not null" json:"name"`
}
