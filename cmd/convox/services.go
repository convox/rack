package main

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "services",
		Description: "manage services",
		Usage:       "",
		Action:      cmdServices,
		Subcommands: []cli.Command{
			{
				Name:            "create",
				Description:     "create a new service",
				Usage:           "<type> [--name=value] [--key-name=value]",
				Action:          cmdServiceCreate,
				SkipFlagParsing: true,
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
			{
				Name:        "link",
				Description: "create a link between a service and an app",
				Usage:       "<name>",
				Action:      cmdLinkCreate,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "unlink",
				Description: "Delete a link between a service and an app",
				Usage:       "<name>",
				Action:      cmdLinkDelete,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdServices(c *cli.Context) {
	services, err := rackClient(c).GetServices()

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
	// ensure type included
	if !(len(c.Args()) > 0) {
		stdcli.Usage(c, "create")
		return
	}

	// ensure everything after type is a flag
	if len(c.Args()) > 1 && !strings.HasPrefix(c.Args()[1], "--") {
		stdcli.Usage(c, "create")
		return
	}

	t := c.Args()[0]
	options := stdcli.ParseOpts(c.Args()[1:])
	for key, value := range options {
		if value == "" {
			options[key] = "true"
		}
	}

	var optionsList []string
	for key, val := range options {
		optionsList = append(optionsList, fmt.Sprintf("%s=%q", key, val))
	}

	if options["name"] == "" {
		options["name"] = fmt.Sprintf("%s-%d", t, (rand.Intn(8999) + 1000))
	}

	fmt.Printf("Creating %s (%s", options["name"], t)
	if len(optionsList) > 0 {
		fmt.Printf(": %s", strings.Join(optionsList, " "))
	}
	fmt.Printf(")... ")

	_, err := rackClient(c).CreateService(t, options)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("CREATING")
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

	if service.Status == "failed" {
		fmt.Printf("Reason  %s\n", service.StatusReason)
	}

	if len(service.Exports) > 0 {
		fmt.Printf("Exports\n")
	}

	for key, value := range service.Exports {
		fmt.Printf("  %s: %s\n", key, value)
	}
}

func cmdLinkCreate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "link")
		return
	}

	name := c.Args()[0]

	_, err = rackClient(c).CreateLink(app, name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Linked %s to %s\n", name, app)
}

func cmdLinkDelete(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "unlink")
		return
	}

	name := c.Args()[0]

	_, err = rackClient(c).DeleteLink(app, name)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Unlinked %s from %s\n", name, app)
}
