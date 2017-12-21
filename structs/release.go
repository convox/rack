package structs

import "time"

type Release struct {
	Id string `json:"id"`

	App      string `json:"app"`
	Build    string `json:"build"`
	Env      string `json:"env"`
	Manifest string `json:"manifest"`
	Status   string `json:"status"`

	Created time.Time `json:"created"`
}

type Releases []Release

type ReleaseCreateOptions struct {
	Build *string
	Env   *string
}

type ReleaseListOptions struct {
	Count *int
}

func NewRelease(app string) *Release {
	return &Release{
		App:     app,
		Created: time.Now(),
		Id:      id("R", 10),
	}
}
