package structs

import "io"

type System struct {
	Count      int               `json:"count"`
	Domain     string            `json:"domain"`
	Name       string            `json:"name"`
	Outputs    map[string]string `json:"outputs,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Provider   string            `json:"provider"`
	Region     string            `json:"region"`
	Status     string            `json:"status"`
	Type       string            `json:"type"`
	Version    string            `json:"version"`
}

type SystemInstallOptions struct {
	Name       *string `flag:"name,n"`
	Output     io.Writer
	Parameters map[string]string
	Version    *string `flag:"version,v"`
}

type SystemProcessesOptions struct {
	All *bool `flag:"all,a" query:"all"`
}

type SystemUninstallOptions struct {
	Force  bool
	Input  io.Reader
	Output io.Writer
}

type SystemUpdateOptions struct {
	Count      *int              `param:"count"`
	Parameters map[string]string `param:"parameters"`
	Type       *string           `param:"type"`
	Version    *string           `param:"version"`
}
