package client

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/convox/rack/api/structs"
)

type Build struct {
	Id       string `json:"id"`
	App      string `json:"app"`
	Logs     string `json:"logs"`
	Manifest string `json:"manifest"`
	Release  string `json:"release"`
	Status   string `json:"status"`

	Description string `json:"description"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`
}

type Builds []Build

func (c *Client) GetBuilds(app string) (Builds, error) {
	var builds Builds

	err := c.Get(fmt.Sprintf("/apps/%s/builds", app), &builds)
	if err != nil {
		return nil, err
	}

	return builds, nil
}

// GetBuildsWithLimit returns a list of the latest builds, with the length specified in limit
func (c *Client) GetBuildsWithLimit(app string, limit int) (Builds, error) {
	var builds Builds

	err := c.Get(fmt.Sprintf("/apps/%s/builds?limit=%d", app, limit), &builds)
	if err != nil {
		return nil, err
	}

	return builds, nil
}

func (c *Client) CreateBuildIndex(app string, index Index, cache bool, manifest string, description string) (*Build, error) {
	var build Build

	data, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"cache":       fmt.Sprintf("%t", cache),
		"description": description,
		"index":       string(data),
		"manifest":    manifest,
	}

	err = c.Post(fmt.Sprintf("/apps/%s/builds", app), params, &build)
	if err != nil {
		return nil, err
	}

	return &build, nil
}

// CreateBuildSource will create a new build from source. If progress of the uploaded is needed, see CreateBuildSourceProgress
func (c *Client) CreateBuildSource(app string, source []byte, cache bool, manifest string, description string) (*Build, error) {
	return c.CreateBuildSourceProgress(app, source, cache, manifest, description, nil)
}

// CreateBuildSourceProgress will create a new build from source with an optional callback to provide progress of the source being uploaded.
func (c *Client) CreateBuildSourceProgress(app string, source []byte, cache bool, manifest string, description string, progressCallback func(s string)) (*Build, error) {
	var build Build

	files := map[string][]byte{
		"source": source,
	}

	params := map[string]string{
		"cache":       fmt.Sprintf("%t", cache),
		"description": description,
		"manifest":    manifest,
	}

	err := c.PostMultipartP(fmt.Sprintf("/apps/%s/builds", app), files, params, &build, progressCallback)
	if err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) CreateBuildUrl(app string, url string, cache bool, manifest string, description string) (*Build, error) {
	var build Build

	params := map[string]string{
		"cache":       fmt.Sprintf("%t", cache),
		"description": description,
		"repo":        url,
		"manifest":    manifest,
	}

	err := c.Post(fmt.Sprintf("/apps/%s/builds", app), params, &build)

	if err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) GetBuild(app, id string) (*Build, error) {
	var build Build

	err := c.Get(fmt.Sprintf("/apps/%s/builds/%s", app, id), &build)
	if err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) StreamBuildLogs(app, id string, output io.WriteCloser) error {
	return c.Stream(fmt.Sprintf("/apps/%s/builds/%s/logs", app, id), nil, nil, output)
}

func (c *Client) CopyBuild(app, id, destApp string) (*Build, error) {
	var build Build

	params := map[string]string{
		"app": destApp,
	}

	err := c.Post(fmt.Sprintf("/apps/%s/builds/%s/copy", app, id), params, &build)

	if err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) DeleteBuild(app, id string) (*Build, error) {
	var build Build

	err := c.Delete(fmt.Sprintf("/apps/%s/builds/%s", app, id), &build)

	return &build, err
}

func (c *Client) UpdateBuild(app, id, manifest, status, reason string) (*Build, error) {
	params := Params{
		"manifest": manifest,
		"status":   status,
		"reason":   reason,
	}

	var build Build

	err := c.Put(fmt.Sprintf("/apps/%s/builds/%s", app, id), params, &build)
	if err != nil {
		return nil, err
	}

	return &build, nil
}

// ExportBuild creats an artifact, representing a build, to be used with another Rack
func (c *Client) ExportBuild(app, id string) ([]byte, error) {

	var buildData []byte
	err := c.Get(fmt.Sprintf("/apps/%s/builds/%s/export", app, id), &buildData)
	if err != nil {
		return nil, err
	}

	return buildData, nil
}

// ImportBuild imports a build artifact
func (c *Client) ImportBuild(app string, source []byte, callback func(s string)) (*structs.Build, error) {

	files := map[string][]byte{
		"source": source,
	}

	params := map[string]string{
		"import": "true",
	}

	build := &structs.Build{}

	err := c.PostMultipartP(fmt.Sprintf("/apps/%s/builds", app), files, params, build, callback)
	if err != nil {
		return nil, err
	}

	return build, nil
}
