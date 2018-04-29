package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "switch",
		Description: "switch to another Convox rack",
		Usage:       "[rack name]",
		ArgsUsage:   "[rack name]",
		Action:      cmdSwitch,
	})
}

func cmdSwitch(c *cli.Context) error {
	if len(c.Args()) < 1 {
		fmt.Println(currentRack(c))
		return nil
	}

	name := c.Args()[0]
	racks := rackList()
	matches := Racks{}

	for _, r := range racks {
		if r.Name == name {
			return switchRack(r.Name)
		}

		if strings.Index(r.Name, name) != -1 {
			matches = append(matches, r)
		}
	}

	if len(matches) > 1 {
		return stdcli.Errorf("ambiguous name: %s", name)
	}

	if len(matches) == 1 {
		return switchRack(matches[0].Name)
	}

	return stdcli.Errorf("could not find rack: %s", name)
}

func switchRack(rack string) error {
	if err := ioutil.WriteFile(filepath.Join(ConfigRoot, "rack"), []byte(rack), 0644); err != nil {
		return err
	}

	fmt.Printf("Switched to %s\n", rack)

	return nil
}
