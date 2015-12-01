package client

import "fmt"

// Mirrors Docker AuthConfiguration
// https://godoc.org/github.com/fsouza/go-dockerclient#AuthConfiguration
type Registry struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Email         string `json:"email,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
}

// Mirrors Docker AuthConfiguration119
//https://godoc.org/github.com/fsouza/go-dockerclient#AuthConfigurations119
type Registries map[string]Registry

func (c *Client) AddRegistry(address, username, password string) (*Registry, error) {
	params := Params{
		"serveraddress": address,
		"username":      username,
		"password":      password,
	}

	var registry Registry

	err := c.Post("/registries", params, &registry)

	if err != nil {
		return nil, err
	}

	return &registry, nil
}

func (c *Client) RemoveRegistry(address string) (*Registry, error) {
	var registry Registry

	err := c.Delete(fmt.Sprintf("/registries/%s", address), &registry)

	if err != nil {
		return nil, err
	}

	return &registry, nil
}

func (c *Client) ListRegistries() (*Registries, error) {
	registries := Registries{}
	err := c.Get("/registries", &registries)

	if err != nil {
		return nil, err
	}

	return &registries, nil
}
