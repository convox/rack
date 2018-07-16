package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
	"github.com/convox/stdsdk"
)

func init() {
	CLI.Command("racks", "list available racks", Racks, stdcli.CommandOptions{
		Validate: stdcli.Args(0),
	})
}

type rack struct {
	Name   string
	Status string
}

func Racks(c *stdcli.Context) error {
	rs, err := racks(c)
	if err != nil {
		return err
	}

	t := c.Table("NAME", "STATUS")

	for _, r := range rs {
		t.AddRow(r.Name, r.Status)
	}

	return t.Print()
}

func racks(c *stdcli.Context) ([]rack, error) {
	rs := []rack{}

	rrs, err := remoteRacks(c)
	if err != nil {
		return nil, err
	}

	rs = append(rs, rrs...)

	lrs, err := localRacks()
	if err != nil {
		return nil, err
	}

	rs = append(rs, lrs...)

	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Name < rs[j].Name
	})

	return rs, nil
}

func remoteRacks(c *stdcli.Context) ([]rack, error) {
	h, err := c.SettingRead("host")
	if err != nil {
		return nil, err
	}

	if h == "" {
		return []rack{}, nil
	}

	racks := []rack{}

	var rs []struct {
		Name         string
		Organization struct {
			Name string
		}
		Status string
	}

	// override local rack to get remote rack list
	endpoint, err := currentEndpoint(c, "")
	if err != nil {
		return nil, err
	}

	p, err := sdk.New(endpoint)
	if err != nil {
		return nil, err
	}

	p.Get("/racks", stdsdk.RequestOptions{}, &rs)

	if rs != nil {
		for _, r := range rs {
			racks = append(racks, rack{
				Name:   fmt.Sprintf("%s/%s", r.Organization.Name, r.Name),
				Status: r.Status,
			})
		}
	}

	return racks, nil
}

func localRacks() ([]rack, error) {
	racks := []rack{}

	data, err := exec.Command("docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").CombinedOutput()
	if err != nil {
		return []rack{}, nil // if no docker then no local racks
	}

	names := strings.Split(strings.TrimSpace(string(data)), "\n")

	for _, name := range names {
		if name == "" {
			continue
		}

		racks = append(racks, rack{
			Name:   fmt.Sprintf("local/%s", name),
			Status: "running",
		})
	}

	return racks, nil
}

func localRackRunning() bool {
	rs, err := localRacks()
	if err != nil {
		return false
	}

	return len(rs) > 0
}

func matchRack(c *stdcli.Context, name string) (*rack, error) {
	rs, err := racks(c)
	if err != nil {
		return nil, err
	}

	matches := []rack{}

	for _, r := range rs {
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
