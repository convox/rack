package main

import (
	"fmt"
	"io/ioutil"
	"os"
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
		rack := currentRack(c)

		if rack == "" {
			fmt.Println("Use `convox racks` to list your available racks and `convox switch <rack>` to select one.")
			os.Exit(1)
		} else {
			fmt.Println(rack)
		}

		return nil
	}

	racks, err := rackClientWithoutLocal(c).Racks()
	if err != nil {
		return stdcli.Error(err)
	}

	rackName := c.Args()[0]
	orgName := ""

	if localRackRunning() && rackName == "local" {
		return switchRack("local")
	}

	parts := strings.Split(rackName, "/")
	if len(parts) == 2 {
		orgName = parts[0]
		rackName = parts[1]
	}

	all := []string{}
	matched := []string{}

	for _, r := range racks {
		rn := fmt.Sprintf("%s/%s", r.Organization.Name, r.Name)
		all = append(all, rn)

		// if no org was specified, collect all the rack name matches
		if orgName == "" {
			if r.Name == rackName {
				matched = append(matched, rn)
			}
		} else {
			if fmt.Sprintf("%s/%s", orgName, rackName) == rn {
				matched = append(matched, rn)
			}
		}
	}

	if len(matched) == 0 {
		errMessages := []string{"Rack not found."}

		if len(all) > 0 {
			errMessages = append(errMessages, ("Try one of the following:\n" + strings.Join(all, "\n")))
		}

		return stdcli.Error(fmt.Errorf(strings.Join(errMessages, " ")))
	}

	if len(matched) > 1 {
		return stdcli.Error(fmt.Errorf("You have access to multiple racks with that name, try one of the following:\n" + strings.Join(matched, "\n")))
	}

	return switchRack(matched[0])
}

func switchRack(rack string) error {
	if err := ioutil.WriteFile(filepath.Join(ConfigRoot, "rack"), []byte(rack), 0644); err != nil {
		return err
	}

	fmt.Printf("Switched to %s\n", rack)

	return nil
}
