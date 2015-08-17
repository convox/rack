package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Service struct {
	Name     string
	Password string
	Type     string
	Status   string
	URL      string

	App string

	Stack string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Services []Service

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "services",
		Description: "manage services",
		Usage:       "",
		Action:      cmdServices,
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new service",
				Usage:       "<name> <postgres|redis>",
				Action:      cmdServiceCreate,
			},
			{
				Name:        "delete",
				Description: "delete a service",
				Usage:       "<name>",
				Action:      cmdServiceDelete,
			},
			{
				Name:        "info",
				Description: "info about a service",
				Usage:       "<name>",
				Action:      cmdServiceInfo,
			},
		},
	})
}

func cmdServices(c *cli.Context) {
	data, err := ConvoxGet("/services")

	if err != nil {
		stdcli.Error(err)
		return
	}

	var services *Services
	err = json.Unmarshal(data, &services)

	if err != nil {
		stdcli.Error(err)
		return
	}

	longest := 7

	for _, service := range *services {
		if len(service.Name) > longest {
			longest = len(service.Name)
		}
	}

	fmt.Printf(fmt.Sprintf("%%-%ds  TYPE\n", longest), "SERVICE")

	for _, service := range *services {
		fmt.Printf(fmt.Sprintf("%%-%ds  %%s\n", longest), service.Name, service.Tags["Service"])
	}
}

func cmdServiceCreate(c *cli.Context) {
	if len(c.Args()) != 2 {
		stdcli.Usage(c, "create")
		return
	}

	name := c.Args()[0]
	t := c.Args()[1]

	fmt.Printf("Creating service %s (%s)... ", name, t)

	v := url.Values{}
	v.Set("name", name)
	v.Set("type", t)
	data, err := ConvoxPostForm("/services", v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err = ConvoxGet("/services/" + name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var s *Service
	err = json.Unmarshal(data, &s)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// poll for complete
	for {
		data, err = ConvoxGet(fmt.Sprintf("/services/%s/status", name))

		if err != nil {
			stdcli.Error(err)
			return
		}

		if string(data) == "running" {
			break
		}

		time.Sleep(3 * time.Second)
	}

	fmt.Printf("OK, %s\n", s.Name)
}

func cmdServiceDelete(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return
	}

	name := c.Args()[0]

	fmt.Printf("Deleting %s... ", name)

	_, err := ConvoxDelete(fmt.Sprintf("/services/%s", name))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}

func cmdServiceInfo(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return
	}

	name := c.Args()[0]

	data, err := ConvoxGet("/services/" + name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var s *Service
	err = json.Unmarshal(data, &s)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Name    %s\n", s.Name)
	fmt.Printf("Status  %s\n", s.Status)
	fmt.Printf("URL     %s\n", s.URL)
}
