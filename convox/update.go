package main

import (
	"fmt"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/inconshreveable/go-update"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/inconshreveable/go-update/check"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "update",
		Description: "update the cli",
		Usage:       "",
		Action:      cmdUpdate,
	})
}

func cmdUpdate(c *cli.Context) {
	params := check.Params{
		AppVersion: Version,
		AppId:      "ap_TKxvw_eIPVyOzl6rKEonCU5DUY",
		Channel:    "stable",
	}

	r, err := params.CheckForUpdate("https://api.equinox.io/1/Updates", update.New())

	if err != nil {
		if err != check.NoUpdateAvailable {
			stdcli.Error(err)
		}
		return
	}

	err, _ = r.Update()

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Updated to %s\n", r.Version)
}
