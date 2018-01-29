package structs

import "io"

type System struct {
	Count      int               `json:"count"`
	Domain     string            `json:"domain"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Outputs    map[string]string `json:"outputs,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
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
	All *bool
}

type SystemUpdateOptions struct {
	InstanceCount *int
	InstanceType  *string
	Output        io.Writer
	Parameters    map[string]string
	Password      *string
	Version       *string
}
