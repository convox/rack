package main

import (
	"fmt"
	"os"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/client"
	"github.com/convox/cli/stdcli"
)

var Version = "dev"

func init() {
	stdcli.VersionPrinter(func(c *cli.Context) {
		fmt.Printf("client: %s\n", c.App.Version)

		system, err := rackClient(c).GetSystem()

		if err != nil {
			stdcli.Error(err)
			return
		}

		host, _, err := currentLogin()

		if err != nil {
			return
		}

		fmt.Printf("server: %s (%s)\n", system.Version, host)
	})
}

func main() {
	app := stdcli.New()
	app.Version = Version
	app.Usage = "command-line application management"

	err := app.Run(os.Args)

	if err != nil {
		os.Exit(1)
	}
}

func rackClient(c *cli.Context) *client.Client {
	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	cl, err := client.New(host, password, c.App.Version)

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	return cl
}

func rackClientManual(host, password, version string) *client.Client {
	cl, err := client.New(host, password, version)

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	return cl
}
