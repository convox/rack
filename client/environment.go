package client

import (
	"fmt"
	"io"
)

type Environment map[string]string

func (c *Client) GetEnvironment(app string) (Environment, error) {
	var env Environment

	err := c.Get(fmt.Sprintf("/apps/%s/environment", app), &env)

	if err != nil {
		return nil, err
	}

	return env, nil
}

func (c *Client) SetEnvironment(app string, body io.Reader) (Environment, error) {
	var env Environment

	err := c.PostBody(fmt.Sprintf("/apps/%s/environment", app), body, &env)

	if err != nil {
		return nil, err
	}

	return env, nil
}

func (c *Client) DeleteEnvironment(app, key string) (Environment, error) {
	var env Environment

	err := c.Delete(fmt.Sprintf("/apps/%s/environment/%s", app, key), &env)

	if err != nil {
		return nil, err
	}

	return env, nil
}

// func (c *Client) CreateApp(name string) (*App, error) {
//   params := Params{
//     "name": name,
//   }

//   var app App

//   err := c.Post("/env", params, &app)

//   if err != nil {
//     return nil, err
//   }

//   return &app, nil
// }

// func (c *Client) GetApp(name string) (*App, error) {
//   var app App

//   err := c.Get(fmt.Sprintf("/env/%s", name), &app)

//   if err != nil {
//     return nil, err
//   }

//   return &app, nil
// }

// func (c *Client) DeleteApp(name string) (*App, error) {
//   var app App

//   err := c.Delete(fmt.Sprintf("/env/%s", name), &app)

//   if err != nil {
//     return nil, err
//   }

//   return &app, nil
// }
