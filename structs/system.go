package structs

import "io"

type System struct {
	Count      int               `json:"count"`
	Domain     string            `json:"domain"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Outputs    map[string]string `json:"outputs,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Provider   string            `json:"provider"`
	Region     string            `json:"region"`
	Status     string            `json:"status"`
	Type       string            `json:"type"`
	Version    string            `json:"version"`
}

type SystemInstallOptions struct {
	Color    *bool
	Output   io.Writer
	Password *string
	Version  *string
}

type SystemProcessesOptions struct {
	All *bool `flag:"all,a" query:"all"`
}

type SystemUninstallOptions struct {
	Color  *bool
	Output io.Writer
}

type SystemUpdateOptions struct {
	Count      *int              `param:"count"`
	Parameters map[string]string `param:"parameters"`
	Type       *string           `param:"type"`
	Version    *string           `param:"version"`
}
