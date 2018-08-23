package structs

type Resource struct {
	Name       string            `json:"name"`
	Parameters map[string]string `json:"parameters"`
	Status     string            `json:"status"`
	Type       string            `json:"type"`
	Url        string            `json:"url"`

	Apps Apps `json:"apps,omitempty"`
}

type Resources []Resource

func (rs Resources) Less(i, j int) bool {
	return rs[i].Name < rs[j].Name
}

type ResourceType struct {
	Name       string             `json:"name"`
	Parameters ResourceParameters `json:"parameters"`
}

type ResourceTypes []ResourceType

func (rts ResourceTypes) Less(i, j int) bool {
	return rts[i].Name < rts[j].Name
}

type ResourceParameter struct {
	Default     string `json:"default"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type ResourceParameters []ResourceParameter

func (rps ResourceParameters) Less(i, j int) bool {
	return rps[i].Name < rps[j].Name
}

type ResourceCreateOptions struct {
	Name       *string           `param:"name"`
	Parameters map[string]string `param:"parameters"`
}

type ResourceUpdateOptions struct {
	Parameters map[string]string `param:"parameters"`
}
