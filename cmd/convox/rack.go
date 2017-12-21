package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/version"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "rack",
		Description: "manage your Convox rack",
		Usage:       "[options]",
		ArgsUsage:   "[subcommand]",
		Action:      cmdRack,
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "logs",
				Description: "stream the rack logs",
				Usage:       "[options]",
				ArgsUsage:   "",
				Action:      cmdRackLogs,
				Flags: []cli.Flag{
					rackFlag,
					cli.StringFlag{
						Name:  "filter",
						Usage: "filter the logs by a given token",
					},
					cli.BoolTFlag{
						Name:  "follow",
						Usage: "keep streaming new log output (default)",
					},
					cli.DurationFlag{
						Name:  "since",
						Usage: "show logs since a duration (e.g. 10m or 1h2m10s)",
						Value: 2 * time.Minute,
					},
				},
			},
			{
				Name:        "params",
				Description: "list advanced rack parameters",
				Usage:       "[options]",
				ArgsUsage:   "[<subcommand>]",
				Action:      cmdRackParams,
				Flags:       []cli.Flag{rackFlag},
				Subcommands: []cli.Command{
					{
						Name:        "set",
						Description: "update advanced rack parameters",
						Usage:       "NAME=VALUE [NAME=VALUE] ...",
						ArgsUsage:   "NAME=VALUE",
						Action:      cmdRackParamsSet,
						Flags: []cli.Flag{rackFlag,
							cli.BoolFlag{
								Name:   "wait",
								EnvVar: "CONVOX_WAIT",
								Usage:  "wait for rack update to finish before returning",
							},
						},
					},
				},
			},
			{
				Name:        "ps",
				Description: "list rack processes",
				Usage:       "[options]",
				ArgsUsage:   "",
				Action:      cmdRackPs,
				Flags: []cli.Flag{
					rackFlag,
					cli.BoolFlag{
						Name:  "stats",
						Usage: "display process cpu/memory stats",
					},
					cli.BoolFlag{
						Name:  "a, all",
						Usage: "display all processes including apps",
					},
				},
			},
			{
				Name:        "scale",
				Description: "scale the rack capacity",
				Usage:       "[options]",
				ArgsUsage:   "",
				Action:      cmdRackScale,
				Flags: []cli.Flag{
					rackFlag,
					cli.IntFlag{
						Name:  "count",
						Usage: "horizontally scale the instance count, e.g. 3 or 10",
					},
					cli.StringFlag{
						Name:  "type",
						Usage: "vertically scale the instance type, e.g. t2.small or c3.xlarge",
					},
				},
			},
			{
				Name:        "update",
				Description: "update rack to the given version",
				Usage:       "[version] [options]",
				ArgsUsage:   "[version]",
				Action:      cmdRackUpdate,
				Flags: []cli.Flag{
					rackFlag,
					cli.BoolFlag{
						Name:   "wait",
						EnvVar: "CONVOX_WAIT",
						Usage:  "wait for rack update to finish before returning",
					},
				},
			},
			{
				Name:        "releases",
				Description: "list a Rack's version history",
				Usage:       "",
				ArgsUsage:   "",
				Action:      cmdRackReleases,
				Flags: []cli.Flag{
					rackFlag,
					cli.BoolFlag{
						Name:  "unpublished",
						Usage: "include unpublished versions",
					},
				},
			},
		},
	})
}

func cmdRack(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	system, err := rackClient(c).GetSystem()
	if err != nil {
		return stdcli.Error(err)
	}

	info := stdcli.NewInfo()

	info.Add("Name", system.Name)
	info.Add("Status", system.Status)
	info.Add("Version", system.Version)

	info.Add("Count", fmt.Sprintf("%d", system.Count))
	if system.Domain != "" {
		info.Add("Domain", system.Domain)
	}
	info.Add("Region", system.Region)
	info.Add("Type", system.Type)

	info.Print()

	return nil
}

func cmdRackLogs(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	err := rackClient(c).StreamRackLogs(c.String("filter"), c.BoolT("follow"), c.Duration("since"), os.Stdout)
	if err != nil {
		return stdcli.Error(err)
	}

	return nil
}

func cmdRackParams(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	system, err := rackClient(c).GetSystem()
	if err != nil {
		return stdcli.Error(err)
	}

	params, err := rackClient(c).ListParameters(system.Name)
	if err != nil {
		return stdcli.Error(err)
	}

	keys := []string{}

	for key := range params {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	t := stdcli.NewTable("NAME", "VALUE")

	for _, key := range keys {
		t.AddRow(key, params[key])
	}

	t.Print()
	return nil
}

func cmdRackParamsSet(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, -1)

	system, err := rackClient(c).GetSystem()
	if err != nil {
		return stdcli.Error(err)
	}

	params := map[string]string{}

	for _, arg := range c.Args() {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return stdcli.Error(fmt.Errorf("invalid argument: %s", arg))
		}

		params[parts[0]] = parts[1]
	}

	stdcli.Startf("Updating parameters")

	err = rackClient(c).SetParameters(system.Name, params)
	if err != nil {
		if strings.Contains(err.Error(), "No updates are to be performed") {
			return stdcli.Error(fmt.Errorf("No updates are to be performed"))
		}
		return stdcli.Error(err)
	}

	stdcli.OK()

	if c.Bool("wait") {
		stdcli.Startf("Waiting for completion")

		// give the rack a few seconds to start updating
		time.Sleep(5 * time.Second)

		if err := waitForRackRunning(c); err != nil {
			return stdcli.Error(err)
		}

		stdcli.OK()
	}

	return nil
}

func cmdRackPs(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	system, err := rackClient(c).GetSystem()
	if err != nil {
		return stdcli.Error(err)
	}

	ps, err := rackClient(c).GetSystemProcesses(structs.SystemProcessesOptions{
		All: options.Bool(c.Bool("all")),
	})
	if err != nil {
		return stdcli.Error(err)
	}

	if c.Bool("stats") {
		fm, err := rackClient(c).ListFormation(system.Name)
		if err != nil {
			return stdcli.Error(err)
		}

		displayProcessesStats(ps, fm, true)
		return nil
	}

	displayProcesses(ps, true)

	return nil
}

func cmdRackUpdate(c *cli.Context) error {
	stdcli.NeedHelp(c)

	// Retrieve list of all versions
	vs, err := version.All()
	if err != nil {
		return stdcli.Error(err)
	}

	// Start by assuming user wants the latest version
	target, err := vs.Latest()
	if err != nil {
		return stdcli.Error(err)
	}

	// if user has provided a version number as an argument, use that instead
	if len(c.Args()) > 0 {
		stdcli.NeedArg(c, 1) // accept no more than one argument
		t, err := vs.Find(c.Args()[0])
		if err != nil {
			return stdcli.Error(err)
		}
		target = t
	}

	system, err := rackClient(c).GetSystem()
	if err != nil {
		return stdcli.Error(err)
	}

	nv, err := vs.Next(system.Version)
	if err != nil && strings.HasSuffix(err.Error(), "is latest") {
		nv = target.Version
	} else if err != nil {
		return stdcli.Error(err)
	}

	next, err := vs.Find(nv)
	if err != nil {
		return stdcli.Error(err)
	}

	// stop at a required release if necessary
	if next.Version < target.Version && next.Required {
		stdcli.Writef("WARNING: Required update found.\nPlease run `convox rack update` again once this update completes.\n")
		target = next
	}

	stdcli.Startf("Updating to <release>%s</release>", target.Version)

	_, err = rackClient(c).UpdateSystem(target.Version)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Wait("UPDATING")

	if c.Bool("wait") {
		stdcli.Startf("Waiting for completion")

		// give the rack a few seconds to start updating
		time.Sleep(5 * time.Second)

		if err := waitForRackRunning(c); err != nil {
			return stdcli.Error(err)
		}

		stdcli.OK()
	}

	return nil
}

func cmdRackScale(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	// initialize to invalid values that indicate no change
	count := -1
	typ := ""

	if c.IsSet("count") {
		count = c.Int("count")
	}

	if c.IsSet("type") {
		typ = c.String("type")
	}

	if count == -1 && typ == "" {
		displaySystem(c)
		return nil
	}

	_, err := rackClient(c).ScaleSystem(count, typ)
	if err != nil {
		return stdcli.Error(err)
	}

	displaySystem(c)
	return nil
}

func cmdRackReleases(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	system, err := rackClient(c).GetSystem()
	if err != nil {
		return stdcli.Error(err)
	}

	pendingVersion := system.Version

	releases, err := rackClient(c).GetSystemReleases()
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("VERSION", "UPDATED", "STATUS")

	for i, r := range releases {
		status := ""

		if system.Status == "updating" && i == 0 {
			pendingVersion = r.Id
			status = "updating"
		}

		if system.Version == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, helpers.HumanizeTime(r.Created), status)
	}

	t.Print()

	next, err := version.Next(system.Version)
	if err != nil {
		return stdcli.Error(err)
	}

	if next > pendingVersion {
		// if strings.Compare(next, pendingVersion) == 1 {
		fmt.Println()
		fmt.Printf("New version available: %s\n", next)
	}

	return nil
}

func displaySystem(c *cli.Context) {
	system, err := rackClient(c).GetSystem()
	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Name     %s\n", system.Name)
	fmt.Printf("Status   %s\n", system.Status)
	fmt.Printf("Version  %s\n", system.Version)
	fmt.Printf("Count    %d\n", system.Count)
	fmt.Printf("Type     %s\n", system.Type)
}

func waitForRackRunning(c *cli.Context) error {
	timeout := time.After(30 * time.Minute)
	tick := time.Tick(2 * time.Second)

	failed := false

	for {
		select {
		case <-tick:
			s, err := rackClient(c).GetSystem()
			if err != nil {
				return err
			}

			switch s.Status {
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
