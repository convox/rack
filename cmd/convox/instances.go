package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "instances",
		Description: "list your Convox rack's instances",
		Usage:       "",
		Action:      cmdInstancesList,
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "keyroll",
				Description: "generate and replace the ec2 keypair used for SSH",
				Usage:       "",
				Action:      cmdInstancesKeyroll,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:            "ssh",
				Description:     "establish secure shell with EC2 instance",
				Usage:           "<id> [command]",
				Action:          cmdInstancesSSH,
				Flags:           []cli.Flag{rackFlag},
				SkipFlagParsing: true,
			},
			{
				Name:        "terminate",
				Description: "terminate an EC2 instance",
				Usage:       "<id>",
				Flags:       []cli.Flag{rackFlag},
				Action:      cmdInstancesTerminate,
			},
		},
	})
}

func cmdInstancesList(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.Error(fmt.Errorf("`convox instances` does not take arguments. Perhaps you meant `convox instances ssh`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	instances, err := rackClient(c).GetInstances()
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("ID", "AGENT", "STATUS", "STARTED", "PS", "CPU", "MEM")

	for _, i := range instances {
		agent := "off"
		if i.Agent {
			agent = "on"
		}

		t.AddRow(i.Id, agent, i.Status,
			helpers.HumanizeTime(i.Started),
			strconv.Itoa(i.Processes),
			fmt.Sprintf("%0.2f%%", i.Cpu*100),
			fmt.Sprintf("%0.2f%%", i.Memory*100))
	}

	t.Print()
	return nil
}

func cmdInstancesKeyroll(c *cli.Context) error {
	fmt.Print("Rolling SSH keys... ")

	err := rackClient(c).InstanceKeyroll()
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("OK")

	return nil
}

func cmdInstancesTerminate(c *cli.Context) error {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "terminate")
		return nil
	}

	id := c.Args()[0]

	fmt.Printf("Terminating %s... ", id)

	err := rackClient(c).TerminateInstance(id)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("OK")
	return nil
}

func cmdInstancesSSH(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "ssh")
		return nil
	}

	id := c.Args()[0]
	cmd := strings.Join(c.Args()[1:], " ")

	code, err := sshWithRestore(c, id, cmd)
	if err != nil {
		return stdcli.Error(err)
	}

	return cli.NewExitError("", code)
}

func sshWithRestore(c *cli.Context, id, cmd string) (int, error) {
	fd := os.Stdin.Fd()
	isTerm := terminal.IsTerminal(int(fd))
	var h, w int

	if isTerm {
		stdinState, err := terminal.GetState(int(fd))
		if err != nil {
			return -1, err
		}

		h, w, err = terminal.GetSize(int(fd))
		if err != nil {
			return -1, err
		}

		defer terminal.Restore(int(fd), stdinState)
	}

	return rackClient(c).SSHInstance(id, cmd, h, w, isTerm, os.Stdin, os.Stdout)
}
