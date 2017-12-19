package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

// ResourceType is the type of an external resource.
type ResourceType struct {
	name, args string
}

var resourceTypes = []ResourceType{
	{
		"fluentd",
		"--url=tcp://fluentd-collector.example.com:24224",
	},
	{
		"memcached",
		"[--instance-type=db.t2.micro] [--num-cache-nodes=1] [--private]",
	},
	{
		"mysql",
		"[--allocated-storage=10] [--database=db-name] [--instance-type=db.t2.micro] [--multi-az] [--password=example] [--private] [--username=example] [--version=5.7.16]",
	},
	{
		"postgres",
		"[--allocated-storage=10] [--backup-retention-period=1] [--database=db-name] [--database-snapshot-identifier=db-snapshot-arn] [--encrypted] [--instance-type=db.t2.micro] [--max-connections={DBInstanceClassMemory/15000000}] [--multi-az] [--password=example] [--private] [--username=example] [--version=9.5.2]",
	},
	{
		"redis",
		"[--automatic-failover-enabled] [--database=db-name] [--encrypted] [--instance-type=cache.t2.micro] [--num-cache-clusters=1] [--private] [--version=3.2.6]",
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
		"webhook",
		"--url=https://console.convox.com/webhooks/1234",
	},
}

var waitSecond = time.Second

func init() {

	usage := "Supported types / options:"
	for _, t := range resourceTypes {
		usage += fmt.Sprintf("\n  %-10s  %s", t.name, t.args)
	}

	stdcli.RegisterCommand(cli.Command{
		Name:        "resources",
		Aliases:     []string{"services"},
		Description: "manage external resources [prev. services]",
		Usage:       "<command> [subcommand] [options] [arguments]",
		ArgsUsage:   "<command>",
		Action:      cmdResources,
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:            "create",
				Description:     "create a new resource",
				Usage:           "<type> [--name=value] [--option-name=value] [options]\n\n" + usage,
				ArgsUsage:       "<type>",
				Action:          cmdResourceCreate,
				Flags:           []cli.Flag{rackFlag, waitFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "delete",
				Description: "delete a resource",
				Usage:       "<name> [options]",
				ArgsUsage:   "<name>",
				Action:      cmdResourceDelete,
				Flags:       []cli.Flag{rackFlag, waitFlag},
			},
			{
				Name:            "update",
				Description:     "update a resource (may cause resource downtime)",
				UsageText:       "update a resource\n\nWARNING: updates may cause resource downtime.",
				Usage:           "<name> --option-name=value [--option-name=value]\n\n" + usage,
				ArgsUsage:       "<name>",
				Action:          cmdResourceUpdate,
				Flags:           []cli.Flag{rackFlag, waitFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "info",
				Description: "info about a resource",
				Usage:       "<name> [options]",
				ArgsUsage:   "<name>",
				Action:      cmdResourceInfo,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "link",
				Description: "create a link between a resource and an app",
				Usage:       "<name> [options]",
				ArgsUsage:   "<name>",
				Action:      cmdLinkCreate,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "unlink",
				Description: "delete a link between a resource and an app",
				Usage:       "<name> [options]",
				ArgsUsage:   "<name>",
				Action:      cmdLinkDelete,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "url",
				Description: "return url for the given resource",
				Usage:       "<name> [options]",
				ArgsUsage:   "<name>",
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
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

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

func checkResourceType(t string) (string, error) {
	for _, resourceType := range resourceTypes {
		if resourceType.name == t {
			return t, nil
		}
	}
	return "", stdcli.Errorf("unsupported resource type %s; see 'convox resources create --help'", t)
}

func cmdResourceCreate(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, -1)

	t, err := checkResourceType(c.Args()[0])
	if err != nil {
		return stdcli.Error(err)
	}
	args := c.Args()[1:]

	// ensure everything after type is a flag
	stdcli.EnsureOnlyFlags(c, args)
	options := stdcli.FlagsToOptions(c, args)

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

	_, err = rackClient(c).CreateResource(t, options)
	if err != nil {
		return stdcli.Error(err)
	}

	return waitForResource(
		rackClient(c),
		options["name"],
		"CREATING",
		c.Bool("wait") || options["wait"] == "true",
	)
}

func cmdResourceUpdate(c *cli.Context) error {
	stdcli.NeedHelp(c)

	name := c.Args()[0]
	args := c.Args()[1:]

	stdcli.EnsureOnlyFlags(c, args)

	options := stdcli.FlagsToOptions(c, args)

	var optionsList []string
	for key, val := range options {
		optionsList = append(optionsList, fmt.Sprintf("%s=%q", key, val))
	}

	optionsSuffix := ""
	if len(optionsList) > 0 {
		optionsSuffix = fmt.Sprintf(" (%s)", strings.Join(optionsList, " "))
	}

	fmt.Printf("Updating %s%s...", name, optionsSuffix)

	_, err := rackClient(c).UpdateResource(name, options)
	if err != nil {
		return stdcli.Error(err)
	}

	return waitForResource(
		rackClient(c),
		options["name"],
		"UPDATING",
		c.Bool("wait") || options["wait"] == "true",
	)
}

func cmdResourceDelete(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	name := c.Args()[0]

	fmt.Printf("Deleting %s... ", name)

	_, err := rackClient(c).DeleteResource(name)
	if err != nil {
		return stdcli.Error(err)
	}

	return waitForResource(
		rackClient(c),
		name,
		"DELETING",
		c.Bool("wait"),
	)
}

func cmdResourceInfo(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

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
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

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
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
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
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
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
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

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

func waitForResource(c *client.Client, n string, t string, w bool) error {
	timeout := time.After(30 * 60 * waitSecond)
	tick := time.Tick(2 * waitSecond)

	if !w {
		fmt.Println(t)
		return nil
	}

	fmt.Println("Waiting for completion")

	// give the rack some time to start updating
	time.Sleep(5 * waitSecond)

	failed := false

	for {
		select {
		case <-tick:
			r, err := c.GetResource(n)
			if err != nil {
				return err
			}

			switch r.Status {
			case "running":
				if failed {
					fmt.Println("DONE")
					return fmt.Errorf("Update rolled back")
				}
				return nil
			case "rollback":
				if !failed {
					failed = true
					fmt.Print("FAILED\nRolling back... ")
				}
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}

	return nil
}
