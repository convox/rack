package main

import (
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "delete",
		Description: "delete apps",
		Action:      cmdDelete,
	})
}

func cmdDelete(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return
	}

	app := c.Args()[0]

	stdcli.Spinner.Prefix = fmt.Sprintf("Deleting %s: ", app)
	stdcli.Spinner.Start()

	_, err := ConvoxDelete(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	stdcli.Spinner.Stop()
	fmt.Printf("\x08\x08OK\n")
}
