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

// NewRelease creates a new release
func NewRelease(app string) *Release {
	return &Release{
		App:     app,
		Created: time.Now(),
		Id:      generateId("R", 10),
	}
}

// Latest returns the latest release determined by the date created.
func (rs Releases) Latest() *Release {
	if len(rs) == 0 {
		return nil
	}

	latest := rs[0]
	for _, r := range rs {
		if latest.Created.Before(r.Created) {
			latest = r
		}
	}

	return &latest
}
