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

	Created time.Time `json:"created"`
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
	Ended    *time.Time
	Logs     *string
	Manifest *string
	Release  *string
	Started  *time.Time
	Status   *string
}

func NewBuild(app string) *Build {
	return &Build{
		App:    app,
		Id:     id("B", 10),
		Status: "created",
		Tags:   map[string]string{},
	}
}
