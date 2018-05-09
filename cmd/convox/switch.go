package main

import (
	"encoding/json"
	"fmt"
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
		rack := currentRack(c)

		if rack == "" {
			return stdcli.Errorf("no rack selected, see a list with `convox racks` then switch to one with `convox switch <rack>`")
		}

		fmt.Println(rack)

		return nil
	}

	name := c.Args()[0]

	r, err := matchRack(name)
	if err != nil {
		return stdcli.Error(err)
	}

	if err := switchRack(*r); err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Switched to %s\n", r.Name)

	return nil
}

func matchRack(name string) (*Rack, error) {
	racks := rackList()
	matches := Racks{}

	for _, r := range racks {
		if r.Name == name {
			return &r, nil
		}

		if strings.Index(r.Name, name) != -1 {
			matches = append(matches, r)
		}
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous rack name: %s", name)
	}

	if len(matches) == 1 {
		return &matches[0], nil
	}

	return nil, fmt.Errorf("could not find rack: %s", name)
}

func switchRack(rack Rack) error {
	if err := writeConfig("rack", rack.Name); err != nil {
		return err
	}

	data, err := json.Marshal(rack)
	if err != nil {
		return err
	}

	if err := writeConfig("switch", string(data)); err != nil {
		return err
	}

	return nil
}
