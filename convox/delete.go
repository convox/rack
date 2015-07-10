package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/Godeps/_workspace/src/golang.org/x/crypto/ssh/terminal"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "delete",
		Description: "delete apps",
		Action:      cmdDelete,
		Flags: []cli.Flag{
			appFlag,
			cli.BoolFlag{
				Name:  "confirm",
				Usage: "confirm deletion. If not specified, prompt for confirmation.",
			},
		},
	})
}

func cmdDelete(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if !c.Bool("confirm") {
		fmt.Printf("Delete '%s'? (y/N) ", app)

		in, err := terminal.ReadPassword(int(os.Stdin.Fd()))

		fmt.Println()

		if err != nil {
			stdcli.Error(err)
			return
		}

		c := strings.ToLower(string(in))

		if !(c == "y" || c == "yes") {
			fmt.Println("Aborting.")
			return
		}
	}

	_, err = ConvoxDelete(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Deleting '%s'\n", app)
}
