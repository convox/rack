package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
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
			"syslog",
			"--url=tcp+tls://logs1.papertrailapp.com:11235",
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
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:            "create",
				Description:     "create a new service.",
				Usage:           "<type> [--name=value] [--option-name=value]\n\n" + usage,
				Action:          cmdServiceCreate,
				Flags:           []cli.Flag{rackFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "delete",
				Description: "delete a service",
				Usage:       "<name>",
				Action:      cmdServiceDelete,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:            "update",
				Description:     "update a service.\n\nWARNING: updates may cause service downtime.",
				Usage:           "<name> --option-name=value [--option-name=value]\n\n" + usage,
				Action:          cmdServiceUpdate,
				Flags:           []cli.Flag{rackFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "info",
				Description: "info about a service.",
				Usage:       "<name>",
				Action:      cmdServiceInfo,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "link",
				Description: "create a link between a service and an app.",
				Usage:       "<name>",
				Action:      cmdLinkCreate,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "unlink",
				Description: "delete a link between a service and an app.",
				Usage:       "<name>",
				Action:      cmdLinkDelete,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "url",
				Description: "return url for the given service",
				Usage:       "<name>",
				Action:      cmdServiceURL,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "proxy",
				Description: "proxy ports from localhost to connect to a service",
				Usage:       "<name>",
				Action:      cmdServiceProxy,
				Flags: []cli.Flag{
					rackFlag,
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

func cmdServices(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox services` does not take arguments. Perhaps you meant `convox services create`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	services, err := rackClient(c).GetServices()
	if err != nil {
		return stdcli.ExitError(err)
	}

	t := stdcli.NewTable("NAME", "TYPE", "STATUS")

	for _, service := range services {
		t.AddRow(service.Name, service.Type, service.Status)
	}

	t.Print()
	return nil
}

func cmdServiceCreate(c *cli.Context) error {
	// ensure type included
	if !(len(c.Args()) > 0) {
		stdcli.Usage(c, "create")
		return nil
	}

	// ensure everything after type is a flag
	if len(c.Args()) > 1 && !strings.HasPrefix(c.Args()[1], "--") {
		stdcli.Usage(c, "create")
		return nil
	}

	t := c.Args()[0]

	if t == "help" || t == "--help" {
		stdcli.Usage(c, "create")
		return nil
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
		return stdcli.ExitError(err)
	}

	fmt.Println("CREATING")
	return nil
}

func cmdServiceUpdate(c *cli.Context) error {
	// ensure name included
	if !(len(c.Args()) > 0) {
		stdcli.Usage(c, "update")
		return nil
	}

	name := c.Args()[0]

	// ensure everything after type is a flag
	if len(c.Args()) > 1 && !strings.HasPrefix(c.Args()[1], "--") {
		stdcli.Usage(c, "update")
		return nil
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
		return nil
	}

	fmt.Printf("Updating %s (%s)...", name, strings.Join(optionsList, " "))

	_, err := rackClient(c).UpdateService(name, options)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("UPDATING")
	return nil
}

func cmdServiceDelete(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	name := c.Args()[0]

	fmt.Printf("Deleting %s... ", name)

	_, err := rackClient(c).DeleteService(name)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("DELETING")
	return nil
}

func cmdServiceInfo(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return nil
	}

	name := c.Args()[0]

	service, err := rackClient(c).GetService(name)
	if err != nil {
		return stdcli.ExitError(err)
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

	return nil
}

func cmdServiceURL(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "url")
		return nil
	}

	name := c.Args()[0]

	service, err := rackClient(c).GetService(name)
	if err != nil {
		return stdcli.ExitError(err)
	}

	if service.Status == "failed" {
		return stdcli.ExitError(fmt.Errorf("Service failure for %s", service.StatusReason))
	}

	if service.URL == "" {
		return stdcli.ExitError(fmt.Errorf("URL does not exist for %s", service.Name))
	}

	fmt.Printf("%s\n", service.URL)

	return nil
}

func cmdLinkCreate(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "link")
		return nil
	}

	name := c.Args()[0]

	_, err = rackClient(c).CreateLink(app, name)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Linked %s to %s\n", name, app)
	return nil
}

func cmdLinkDelete(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "unlink")
		return nil
	}

	name := c.Args()[0]

	_, err = rackClient(c).DeleteLink(app, name)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Unlinked %s from %s\n", name, app)
	return nil
}

func cmdServiceProxy(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return nil
	}

	name := c.Args()[0]

	service, err := rackClient(c).GetService(name)
	if err != nil {
		return stdcli.ExitError(err)
	}

	export, ok := service.Exports["URL"]
	if !ok {
		return stdcli.ExitError(fmt.Errorf("%s does not expose a URL", name))
	}

	u, err := url.Parse(export)
	if err != nil {
		return stdcli.ExitError(err)
	}

	remotehost, remoteport, err := net.SplitHostPort(u.Host)
	if err != nil {
		return stdcli.ExitError(err)
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
		return stdcli.ExitError(err)
	}

	rp, err := strconv.Atoi(remoteport)
	if err != nil {
		return stdcli.ExitError(err)
	}

	proxy(localhost, lp, remotehost, rp, rackClient(c))
	return nil
}
