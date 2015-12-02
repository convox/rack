package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "instances",
		Description: "list your Convox rack's instances",
		Usage:       "",
		Action:      cmdInstancesList,
		Subcommands: []cli.Command{
			{
				Name:        "ssh",
				Description: "establish secure shell with EC2 instance",
				Usage:       "<id>",
				Action:      cmdInstancesSSH,
			},
			{
				Name:        "terminate",
				Description: "terminate an EC2 instance",
				Usage:       "<id>",
				Action:      cmdInstancesTerminate,
			},
		},
	})
}

func cmdInstancesList(c *cli.Context) {
	instances, err := rackClient(c).GetInstances()

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("ID", "IP", "STATUS", "PROCESSES", "CPU", "MEM")

	for _, i := range instances {
		t.AddRow(i.Id, i.Ip, i.Status, strconv.Itoa(i.Processes),
			fmt.Sprintf("%0.2f%%", i.Cpu*100),
			fmt.Sprintf("%0.2f%%", i.Memory*100))
	}
	t.Print()
}

func cmdInstancesTerminate(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "terminate")
		return
	}

	id := c.Args()[0]
	err := rackClient(c).TerminateInstance(id)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Successfully sent terminate to instance %q\n", id)
}

func cmdInstancesSSH(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "ssh")
		return
	}

	id := c.Args()[0]
	err := rackClient(c).SSHInstance(id, os.Stdin, os.Stdout)

	if err != nil {
		stdcli.Error(err)
		return
	}
}
