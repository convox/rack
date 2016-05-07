package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

type ServiceType struct {
	name, args string
}

func init() {
	types := []ServiceType{
		ServiceType{
			"mysql",
			"[--allocated-storage=10] [--instance-type=db.t2.micro] [--multi-az] [--private]",
		},
		ServiceType{
			"papertrail",
			"--url=logs1.papertrailapp.com:11235",
		},
		ServiceType{
			"postgres",
			"[--allocated-storage=10] [--instance-type=db.t2.micro] [--max-connections={DBInstanceClassMemory/15000000}] [--multi-az] [--private]",
		},
		ServiceType{
			"redis",
			"[--automatic-failover-enabled] [--instance-type=cache.t2.micro] [--num-cache-clusters=1] [--private]",
		},
		ServiceType{
			"s3",
			"[--topic=sns-service-name] [--versioning]",
		},
		ServiceType{
			"sns",
			"[--queue=sqs-service-name]",
		},
		ServiceType{
			"sqs",
			"",
		},
		ServiceType{
			"webhook",
			"--url=https://console.convox.com/webhooks/1234",
		},
	}

	usage := "Supported types / options:"
	for _, t := range types {
		usage += fmt.Sprintf("\n  %-10s  %s", t.name, t.args)
	}

	stdcli.RegisterCommand(cli.Command{
		Name:        "services",
		Description: "manage services",
		Usage:       "",
		Action:      cmdServices,
		Subcommands: []cli.Command{
			{
				Name:            "create",
				Description:     "create a new service.",
				Usage:           "<type> [--name=value] [--option-name=value]\n\n" + usage,
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
				Name:            "update",
				Description:     "update a service.\n\nWARNING: updates may cause service downtime.",
				Usage:           "<name> --option-name=value [--option-name=value]\n\n" + usage,
				Action:          cmdServiceUpdate,
				SkipFlagParsing: true,
			},
			{
				Name:        "info",
				Description: "info about a service.",
				Usage:       "<name>",
				Action:      cmdServiceInfo,
			},
			{
				Name:        "link",
				Description: "create a link between a service and an app.",
				Usage:       "<name>",
				Action:      cmdLinkCreate,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "unlink",
				Description: "delete a link between a service and an app.",
				Usage:       "<name>",
				Action:      cmdLinkDelete,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "proxy",
				Description: "proxy ports from localhost to connect to a service",
				Usage:       "<name>",
				Action:      cmdServiceProxy,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "listen, l",
						Value: "",
						Usage: "[[addr:]port]",
					},
				},
			},
		},
	})
}

func cmdServices(c *cli.Context) {
	if len(c.Args()) > 0 {
		stdcli.Error(fmt.Errorf("`convox services` does not take arguments. Perhaps you meant `convox services create`?"))
		return
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return
	}

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

	if t == "help" {
		stdcli.Usage(c, "create")
		return
	}

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

func cmdServiceUpdate(c *cli.Context) {
	// ensure name included
	if !(len(c.Args()) > 0) {
		stdcli.Usage(c, "update")
		return
	}

	name := c.Args()[0]

	// ensure everything after type is a flag
	if len(c.Args()) > 1 && !strings.HasPrefix(c.Args()[1], "--") {
		stdcli.Usage(c, "update")
		return
	}

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

	if len(optionsList) == 0 {
		stdcli.Usage(c, "update")
		return
	}

	fmt.Printf("Updating %s (%s)...", name, strings.Join(optionsList, " "))

	_, err := rackClient(c).UpdateService(name, options)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("UPDATING")
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

		for key, value := range service.Exports {
			fmt.Printf("  %s: %s\n", key, value)
		}
	} else if service.URL != "" {
		// NOTE: this branch is deprecated
		fmt.Printf("URL     %s\n", service.URL)
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

func cmdServiceProxy(c *cli.Context) {
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

	export, ok := service.Exports["URL"]

	if !ok {
		stdcli.Error(fmt.Errorf("%s does not expose a URL", name))
		return
	}

	u, err := url.Parse(export)

	if err != nil {
		stdcli.Error(err)
		return
	}

	remotehost, remoteport, err := net.SplitHostPort(u.Host)

	if err != nil {
		stdcli.Error(err)
		return
	}

	localhost := "127.0.0.1"
	localport := remoteport

	if listen := c.String("listen"); listen != "" {
		parts := strings.Split(listen, ":")

		switch len(parts) {
		case 1:
			localport = parts[0]
		case 2:
			localhost = parts[0]
			localport = parts[1]
		}
	}

	lp, err := strconv.Atoi(localport)

	if err != nil {
		stdcli.Error(err)
		return
	}

	rp, err := strconv.Atoi(remoteport)

	if err != nil {
		stdcli.Error(err)
		return
	}

	proxy(localhost, lp, remotehost, rp, rackClient(c))
}
