package structs

type Resource struct {
	Name       string            `json:"name"`
	Parameters map[string]string `json:"parameters"`
	Status     string            `json:"status"`
	Type       string            `json:"type"`
	Url        string            `json:"url"`

	Apps Apps `json:"-"`
}

type Resources []Resource

type ResourceCreateOptions struct {
	Parameters map[string]string
}
