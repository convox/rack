package client

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/convox/rack/structs"
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

func (c *Client) CreateBuildIndex(app string, index Index, cache bool, config string, description string) (*Build, error) {
	var build Build

	data, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"cache":       fmt.Sprintf("%t", cache),
		"config":      config,
		"description": description,
		"index":       string(data),
	}

	system, err := c.GetSystem()
	if err != nil {
		return nil, err
	}

	// backwards compatible
	if system.Version < "20160928105531" {
		params["manifest"] = params["config"]
	}

	err = c.Post(fmt.Sprintf("/apps/%s/builds", app), params, &build)
	if err != nil {
		return nil, err
	}

	return &build, nil
}

type CreateBuildSourceOptions struct {
	Cache       bool
	Config      string
	Description string
	Progress    Progress
}

// CreateBuildSource will create a new build from source
func (c *Client) CreateBuildSource(app string, source io.Reader, opts CreateBuildSourceOptions) (*Build, error) {
	var build Build

	popts := PostMultipartOptions{
		Files: Files{
			"source": source,
		},
		Params: map[string]string{
			"cache":       fmt.Sprintf("%t", opts.Cache),
			"config":      opts.Config,
			"description": opts.Description,
		},
		Progress: opts.Progress,
	}

	system, err := c.GetSystem()
	if err != nil {
		return nil, err
	}

	// backwards compatible
	if system.Version < "20160928105531" {
		popts.Params["manifest"] = popts.Params["config"]
	}

	if err := c.PostMultipart(fmt.Sprintf("/apps/%s/builds", app), popts, &build); err != nil {
		return nil, err
	}

	return &build, nil
}

func (c *Client) CreateBuildUrl(app string, url string, cache bool, config string, description string) (*Build, error) {
	var build Build

	params := map[string]string{
		"cache":       fmt.Sprintf("%t", cache),
		"description": description,
		"config":      config,
		"url":         url,
	}

	system, err := c.GetSystem()
	if err != nil {
		return nil, err
	}

	// backwards compatible
	if system.Version < "20160928105531" {
		params["manifest"] = params["config"]
		params["repo"] = params["url"]
	}

	err = c.Post(fmt.Sprintf("/apps/%s/builds", app), params, &build)
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

func (c *Client) StreamBuildLogs(app, id string, output io.Writer) error {
	system, err := c.GetSystem()
	if err != nil {
		return err
	}

	// backwards compatible
	if system.Version < "20160928105531" {
		build, err := c.GetBuild(app, id)
		if err != nil {
			return err
		}

		_, err = output.Write([]byte(build.Logs))
		if err != nil {
			return err
		}

		return nil
	}

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
func (c *Client) ExportBuild(app, id string, w io.Writer) error {
	return c.Get(fmt.Sprintf("/apps/%s/builds/%s.tgz", app, id), w)
}

type ImportBuildOptions struct {
	Progress Progress
}

// ImportBuild imports a build artifact
func (c *Client) ImportBuild(app string, r io.Reader, opts ImportBuildOptions) (*structs.Build, error) {
	system, err := c.GetSystem()
	if err != nil {
		return nil, err
	}

	// backwards compatible
	if system.Version < "20160928105531" {
		return c.importBuildClassic(app, r, opts)
	}

	popts := PostMultipartOptions{
		Files: map[string]io.Reader{
			"image": r,
		},
		Progress: opts.Progress,
	}

	build := &structs.Build{}

	if err := c.PostMultipart(fmt.Sprintf("/apps/%s/builds", app), popts, &build); err != nil {
		return nil, err
	}

	return build, nil
}

func (c *Client) importBuildClassic(app string, r io.Reader, opts ImportBuildOptions) (*structs.Build, error) {
	popts := PostMultipartOptions{
		Files: map[string]io.Reader{
			"source": r,
		},
		Params: map[string]string{
			"import": "true",
		},
		Progress: opts.Progress,
	}

	build := &structs.Build{}

	if err := c.PostMultipart(fmt.Sprintf("/apps/%s/builds", app), popts, &build); err != nil {
		return nil, err
	}

	return build, nil
}
