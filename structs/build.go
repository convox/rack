package structs

import (
	"time"
)

type Build struct {
	Id          string `json:"id"`
	App         string `json:"app"`
	Description string `json:"description"`
	Logs        string `json:"logs"`
	Manifest    string `json:"manifest"`
	Process     string `json:"process"`
	Release     string `json:"release"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`

	Tags map[string]string `json:"-"`
}

type Builds []Build

type BuildListOptions struct {
	Count *int
}

type BuildCreateOptions struct {
	Cache       *bool
	Config      *string
	Description *string
	Development *bool
	Manifest    *string
}

type BuildUpdateOptions struct {
	Ended    *time.Time `param:"ended"`
	Logs     *string    `param:"logs"`
	Manifest *string    `param:"manifest"`
	Release  *string    `param:"release"`
	Started  *time.Time `param:"started"`
	Status   *string    `param:"status"`
}

func NewBuild(app string) *Build {
	return &Build{
		App:    app,
		Id:     id("B", 10),
		Status: "created",
		Tags:   map[string]string{},
	}
}
