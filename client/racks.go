package client

type Rack struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (c *Client) Racks() (racks []Rack, err error) {
	err = c.Get("/racks", &racks)
	return racks, err
}
