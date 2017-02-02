package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

// ResourceType is the type of an external resource.
type ResourceType struct {
	name, args string
}

func init() {
	types := []ResourceType{
		{
			"memcached",
			"[--instance-type=db.t2.micro] [--num-cache-nodes=1] [--private]",
		},
		{
			"mysql",
			"[--allocated-storage=10] [--database=db-name] [--instance-type=db.t2.micro] [--multi-az] [--password=example] [--private] [--username=example]",
		},
		{
			"postgres",
			"[--allocated-storage=10] [--database=db-name] [--instance-type=db.t2.micro] [--max-connections={DBInstanceClassMemory/15000000}] [--multi-az] [--password=example] [--private] [--username=example] [--version=9.5.2]",
		},
		{
			"redis",
			"[--automatic-failover-enabled] [--database=db-name] [--instance-type=cache.t2.micro] [--num-cache-clusters=1] [--private]",
		},
		{
			"s3",
			"[--topic=sns-topic-name] [--versioning]",
		},
		{
			"sns",
			"[--queue=sqs-queue-name]",
		},
		{
			"sqs",
			"[--message-retention-period=345600] [--receive-message-wait-time=0] [--visibility-timeout=30]",
		},
		{
			"syslog",
			"--url=tcp+tls://logs1.papertrailapp.com:11235 [--private]",
		},
		{
			"fluentd",
			"--url=tcp://fluentd-collector.example.com:24224",
		},
		{
			"webhook",
			"--url=https://console.convox.com/webhooks/1234",
		},
	}

	usage := "Supported types / options:"
	for _, t := range types {
		usage += fmt.Sprintf("\n  %-10s  %s", t.name, t.args)
	}

	stdcli.RegisterCommand(cli.Command{
		Name:        "resources",
		Aliases:     []string{"services"},
		Description: "manage external resources [prev. services]",
		Usage:       "",
		Action:      cmdResources,
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:            "create",
				Description:     "create a new resource.",
				Usage:           "<type> [--name=value] [--option-name=value]\n\n" + usage,
				Action:          cmdResourceCreate,
				Flags:           []cli.Flag{rackFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "delete",
				Description: "delete a resource",
				Usage:       "<name>",
				Action:      cmdResourceDelete,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:            "update",
				Description:     "update a resource.\n\nWARNING: updates may cause resource downtime.",
				Usage:           "<name> --option-name=value [--option-name=value]\n\n" + usage,
				Action:          cmdResourceUpdate,
				Flags:           []cli.Flag{rackFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "info",
				Description: "info about a resource.",
				Usage:       "<name>",
				Action:      cmdResourceInfo,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "link",
				Description: "create a link between a resource and an app.",
				Usage:       "<name>",
				Action:      cmdLinkCreate,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "unlink",
				Description: "delete a link between a resource and an app.",
				Usage:       "<name>",
				Action:      cmdLinkDelete,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "url",
				Description: "return url for the given resource",
				Usage:       "<name>",
				Action:      cmdResourceURL,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "proxy",
				Description: "proxy ports from localhost to connect to a resource",
				Usage:       "<name>",
				Action:      cmdResourceProxy,
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

func cmdResources(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.Error(fmt.Errorf("`convox resources` does not take arguments. Perhaps you meant `convox resources create`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	resources, err := rackClient(c).GetResources()
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("NAME", "TYPE", "STATUS")

	for _, resource := range resources {
		t.AddRow(resource.Name, resource.Type, resource.Status)
	}

	t.Print()
	return nil
}

func cmdResourceCreate(c *cli.Context) error {
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

	if t == "help" || t == "--help" || t == "-h" {
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

	// special cases
	switch {
	case t == "postgres" && options["version"] != "":
		parts := strings.Split(options["version"], ".")
		if len(parts) < 3 {
			return stdcli.Error(fmt.Errorf("invalid version: %s", options["version"]))
		}
		options["family"] = fmt.Sprintf("postgres%s.%s", parts[0], parts[1])
	}

	fmt.Printf("Creating %s (%s", options["name"], t)
	if len(optionsList) > 0 {
		sort.Strings(optionsList)
		fmt.Printf(": %s", strings.Join(optionsList, " "))
	}
	fmt.Printf(")... ")

	_, err := rackClient(c).CreateResource(t, options)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("CREATING")
	return nil
}

func cmdResourceUpdate(c *cli.Context) error {
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

	_, err := rackClient(c).UpdateResource(name, options)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("UPDATING")
	return nil
}

func cmdResourceDelete(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	name := c.Args()[0]

	fmt.Printf("Deleting %s... ", name)

	_, err := rackClient(c).DeleteResource(name)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("DELETING")
	return nil
}

func cmdResourceInfo(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return nil
	}

	name := c.Args()[0]

	resource, err := rackClient(c).GetResource(name)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Name    %s\n", resource.Name)
	fmt.Printf("Status  %s\n", resource.Status)

	if resource.Status == "failed" {
		fmt.Printf("Reason  %s\n", resource.StatusReason)
	}

	if len(resource.Exports) > 0 {
		fmt.Printf("Exports\n")

		for key, value := range resource.Exports {
			fmt.Printf("  %s: %s\n", key, value)
		}
	} else if resource.URL != "" {
		// NOTE: this branch is deprecated
		fmt.Printf("URL     %s\n", resource.URL)
	}

	return nil
}

func cmdResourceURL(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "url")
		return nil
	}

	name := c.Args()[0]

	resource, err := rackClient(c).GetResource(name)
	if err != nil {
		return stdcli.Error(err)
	}

	if resource.Status == "failed" {
		return stdcli.Error(fmt.Errorf("Resource failure for %s", resource.StatusReason))
	}

	if resource.URL == "" {
		return stdcli.Error(fmt.Errorf("URL does not exist for %s", resource.Name))
	}

	fmt.Printf("%s\n", resource.URL)

	return nil
}

func cmdLinkCreate(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "link")
		return nil
	}

	name := c.Args()[0]

	_, err = rackClient(c).CreateLink(app, name)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Linked %s to %s\n", name, app)
	return nil
}

func cmdLinkDelete(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "unlink")
		return nil
	}

	name := c.Args()[0]

	_, err = rackClient(c).DeleteLink(app, name)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Unlinked %s from %s\n", name, app)
	return nil
}

func cmdResourceProxy(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "proxy")
		return nil
	}

	name := c.Args()[0]

	resource, err := rackClient(c).GetResource(name)
	if err != nil {
		return stdcli.Error(err)
	}

	export, ok := resource.Exports["URL"]
	if !ok {
		return stdcli.Error(fmt.Errorf("%s does not expose a URL", name))
	}

	u, err := url.Parse(export)
	if err != nil {
		return stdcli.Error(err)
	}

	remotehost, remoteport, err := net.SplitHostPort(u.Host)
	if err != nil {
		return stdcli.Error(err)
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
		return stdcli.Error(err)
	}

	rp, err := strconv.Atoi(remoteport)
	if err != nil {
		return stdcli.Error(err)
	}

	proxy(localhost, lp, remotehost, rp, rackClient(c))
	return nil
}
