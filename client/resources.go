package client

import "fmt"

// Resource is an external resource.
type Resource struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	StatusReason string            `json:"status-reason"`
	Type         string            `json:"type"`
	Exports      map[string]string `json:"exports"`
	// DEPRECATED: should inject any data in Exports
	// we only set this on the outgoing response for old clients
	URL string `json:"url"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

// Resources is a list of resources.
type Resources []Resource

// GetResources retrieves a list of resources.
func (c *Client) GetResources() (Resources, error) {
	var resources Resources

	err := c.Get("/resources", &resources)

	if err != nil {
		return nil, err
	}

	return resources, nil
}

// CreateResource creates a new resource.
func (c *Client) CreateResource(kind string, options map[string]string) (*Resource, error) {
	params := Params(options)
	params["type"] = kind
	var resource Resource

	err := c.Post("/resources", params, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// GetResource retrieves a resource by name.
func (c *Client) GetResource(name string) (*Resource, error) {
	var resource Resource

	err := c.Get(fmt.Sprintf("/resources/%s", name), &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// DeleteResource deletes a resource.
func (c *Client) DeleteResource(name string) (*Resource, error) {
	var resource Resource

	err := c.Delete(fmt.Sprintf("/resources/%s", name), &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// UpdateResource updates a resource.
func (c *Client) UpdateResource(name string, options map[string]string) (*Resource, error) {
	params := Params(options)
	var resource Resource

	err := c.Put(fmt.Sprintf("/resources/%s", name), params, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
