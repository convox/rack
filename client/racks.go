package client

type Organization struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Rack struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	Organization *Organization `json:"organization"`
}

func (c *Client) Racks() (racks []Rack, err error) {
	err = c.Get("/racks", &racks)
	return racks, err
}
