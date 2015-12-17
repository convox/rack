package client

import "fmt"

type Service struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Size   string `json:"size"`
}

type Services []Service

func (c *Client) GetServices() (Services, error) {
	var services Services

	err := c.Get("/services", &services)

	if err != nil {
		return nil, err
	}

	return services, nil
}

func (c *Client) CreateService(kind, name string, options map[string]string) (*Service, error) {

	params := Params(options)
	//NOTE: might move this to ParseOpts
	for key, value := range params {
		if value == "" {
			params[key] = "true"
		}
	}

	params["name"] = name
	params["type"] = kind
	var service Service

	err := c.Post("/services", params, &service)

	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (c *Client) GetService(name string) (*Service, error) {
	var service Service

	err := c.Get(fmt.Sprintf("/services/%s", name), &service)

	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (c *Client) DeleteService(name string) (*Service, error) {
	var service Service

	err := c.Delete(fmt.Sprintf("/services/%s", name), &service)

	if err != nil {
		return nil, err
	}

	return &service, nil
}
