package main

import (
	"fmt"
	"time"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
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
				Usage:       "<type> <name>",
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
	services, err := rackClient(c).ListServices()

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("NAME", "TYPE", "STATUS")

	for _, service := range services {
		t.AddRow(service.Name, service.Type, service.Status)
	}

	t.Print()
}

func cmdServiceCreate(c *cli.Context) {
	if len(c.Args()) != 2 {
		stdcli.Usage(c, "create")
		return
	}

	t := c.Args()[0]
	name := c.Args()[1]

	fmt.Printf("Creating %s (%s)... ", name, t)

	service, err := rackClient(c).CreateService(t, name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	for {
		s, err := rackClient(c).GetService(service.Name)

		if err != nil {
			stdcli.Error(err)
			return
		}

		if s.Status == "running" {
			break
		}

		time.Sleep(3 * time.Second)
	}

	fmt.Println("OK")
}

func cmdServiceDelete(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return
	}

	name := c.Args()[0]

	fmt.Printf("Deleting %s... ", name)

	_, err := rackClient(c).DeleteService(name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("DELETING")
}

func cmdServiceInfo(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return
	}

	name := c.Args()[0]

	service, err := rackClient(c).GetService(name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Name    %s\n", service.Name)
	fmt.Printf("Status  %s\n", service.Status)
	fmt.Printf("URL     %s\n", service.URL)
}
