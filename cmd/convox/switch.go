package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "switch",
		Description: "switch to another Convox rack",
		Usage:       "[rack name]",
		Action:      cmdSwitch,
	})
}

func cmdSwitch(c *cli.Context) error {
	if len(c.Args()) < 1 {
		rack := currentRack(c)

		if rack == "" {
			fmt.Println("Use `convox racks` to list your available racks and `convox switch <rack>` to select one.")
			os.Exit(1)
		} else {
			fmt.Println(rack)
		}

		return nil
	}

	rack := c.Args()[0]

	racks, err := rackClient(c).Racks()

	if err != nil {
		return stdcli.ExitError(err)
	}

	found := false

	for _, r := range racks {
		if fmt.Sprintf("%s/%s", r.Organization.Name, r.Name) == rack {
			found = true
			break
		}
	}

	if !found {
		return stdcli.ExitError(fmt.Errorf("no such rack: %s", rack))
	}

	if err := ioutil.WriteFile(filepath.Join(ConfigRoot, "rack"), []byte(rack), 0644); err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Switched to %s\n", rack)

	return nil
}
