package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "racks",
		Description: "list your Convox racks",
		Usage:       "",
		Action:      cmdRacks,
		Subcommands: []cli.Command{
			{
				Name:        "known",
				Description: "list racks used by this workstation",
				Usage:       "",
				Action:      cmdKnownRacks,
			},
		},
	})
}

func cmdRacks(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox racks` does not take arguments. Perhaps you meant `convox racks`?"))
	}

	racks, err := rackClient(c).Racks()
	if err != nil {
		return stdcli.ExitError(err)
	}

	t := stdcli.NewTable("RACK", "STATUS")
	for _, rack := range racks {
		name := rack.Name
		if rack.Organization != nil {
			name = fmt.Sprintf("%s/%s", rack.Organization.Name, name)
		}
		t.AddRow(name, rack.Status)
	}
	t.Print()
	return nil
}

func cmdKnownRacks(c *cli.Context) error {
	config := filepath.Join(ConfigRoot, "auth")
	data, _ := ioutil.ReadFile(filepath.Join(config))
	if data == nil {
		data = []byte("{}")
	}

	var auth ConfigAuth
	err := json.Unmarshal(data, &auth)

	if err != nil {
		return err
	}

	t := stdcli.NewTable("RACK")
	for rack := range auth {
		t.AddRow(rack)
	}

	t.Print()
	return nil
}
