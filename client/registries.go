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
		"username":      username,
		"password":      password,
		"email":         email,
		"serveraddress": server,
	}

	system, err := c.GetSystem()
	if err != nil {
		return nil, err
	}

	// backwards compatible
	// FIXME: pick the right version here
	if system.Version < "30000000000000" {
		params["server"] = params["serveraddress"]
		delete(params, "serveraddress")
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
