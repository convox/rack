package main

import (
	"encoding/json"
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

	r, err := matchRack(name)
	if err != nil {
		return stdcli.Error(err)
	}

	return switchRack(*r)
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
	if err := ioutil.WriteFile(filepath.Join(ConfigRoot, "rack"), []byte(rack.Name), 0644); err != nil {
		return err
	}

	data, err := json.Marshal(rack)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(ConfigRoot, "switch"), data, 0644); err != nil {
		return err
	}

	fmt.Printf("Switched to %s\n", rack.Name)

	return nil
}
