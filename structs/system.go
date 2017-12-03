package structs

type System struct {
	Count      int               `json:"count"`
	Domain     string            `json:"domain"`
	Name       string            `json:"name"`
	Outputs    map[string]string `json:"outputs"`
	Parameters map[string]string `json:"parameters"`
	Region     string            `json:"region"`
	Status     string            `json:"status"`
	Type       string            `json:"type"`
	Version    string            `json:"version"`
}

type SystemProcessesOptions struct {
	All bool
}

type SystemUpdateOptions struct {
	Parameters map[string]string
}
