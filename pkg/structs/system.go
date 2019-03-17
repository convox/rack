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
	Id         *string
	Name       *string `flag:"name,n"`
	Parameters map[string]string
	Raw        *bool   `flag:"raw"`
	Version    *string `flag:"version,v"`
}

type SystemProcessesOptions struct {
	All *bool `flag:"all,a" query:"all"`
}

type SystemUninstallOptions struct {
	Force *bool `flag:"force,f"`
	Input io.Reader
}

type SystemUpdateOptions struct {
	Count      *int              `param:"count"`
	Parameters map[string]string `param:"parameters"`
	Type       *string           `param:"type"`
	Version    *string           `param:"version"`
}
