package structs

// Resource is a dependency of an app that communicates with it over the network.
// Documentation: https://convox.com/docs/about-resources/
type Resource struct {
	Name         string `json:"name"`
	Stack        string `json:"-"`
	Status       string `json:"status"`
	StatusReason string `json:"status-reason"`
	Type         string `json:"type"`

	Apps    Apps              `json:"apps"`
	Exports map[string]string `json:"exports"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

// Resources is a list of resources.
type Resources []Resource
