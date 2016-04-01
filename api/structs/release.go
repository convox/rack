package structs

import "time"

type Release struct {
	Id       string    `json:"id"`
	App      string    `json:"app"`
	Build    string    `json:"build"`
	Env      string    `json:"env"`
	Manifest string    `json:"manifest"`
	Created  time.Time `json:"created"`
}

type Releases []Release

func NewRelease(app string) *Release {
	return &Release{
		App: app,
		Id:  generateId("R", 10),
	}
}
