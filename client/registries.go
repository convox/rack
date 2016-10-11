package client

import "fmt"

type Registry struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Registries []Registry

func (c *Client) AddRegistry(server, username, password, email string) (*Registry, error) {
	params := Params{
		"username": username,
		"password": password,
		"email":    email,
		"server":   server,
	}

	system, err := c.GetSystem()
	if err != nil {
		return nil, err
	}

	// backwards compatible
	if system.Version < "20161006183008" {
		params["serveraddress"] = params["server"]
		delete(params, "server")
	}

	var registry Registry

	err = c.Post("/registries", params, &registry)
	if err != nil {
		return nil, err
	}

	return &registry, nil
}

func (c *Client) RemoveRegistry(server string) error {
	err := c.Delete(fmt.Sprintf("/registries?server=%s", server), nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ListRegistries() (*Registries, error) {
	registries := Registries{}

	if err := c.Get("/registries", &registries); err != nil {
		return nil, err
	}

	return &registries, nil
}
