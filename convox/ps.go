package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Process struct {
	App         string
	Command     string
	CPU         int64
	Id          string
	Memory      int64
	Name        string
	Release     string
	ServiceType string
}

type Processes []Process

type ProcessTop struct {
	Titles    []string
	Processes [][]string
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:			"ps",
		Description:	"list an app's processes",
		Usage:       	"",
		Action:      	cmdPs,
		Flags: 			[]cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:        "stop",
				Description: "stop a process",
				Usage:       "id",
				Action:      cmdPsStop,
				Flags: []cli.Flag{appFlag},
			},
		},
	})
}

func cmdPs(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) == 0 {
		processList(app)
	} else {
		processTop(app, c.Args()[0])
	}
}

func cmdPsStop(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "stop")
		return
	}

	id := c.Args()[0]

	_, err = ConvoxDelete(fmt.Sprintf("/apps/%s/processes/%s", app, id))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Stopping %s\n", id)
}

func processList(app string) {
	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/processes", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var processes *Processes
	err = json.Unmarshal(data, &processes)

	if err != nil {
		stdcli.Error(err)
		return
	}

	longest := 7

	for _, ps := range *processes {
		if len(ps.Name) > longest {
			longest = len(ps.Name)
		}
	}

	fmt.Printf(fmt.Sprintf("%%-12s  %%-%ds  %%-11s  %%-5s  %%s\n", longest), "ID", "PROCESS", "RELEASE", "MEM", "COMMAND")

	for _, ps := range *processes {
		fmt.Printf(fmt.Sprintf("%%-12s  %%-%ds  %%-11s  %%-5d  %%s\n", longest), ps.Id, ps.Name, ps.Release, ps.Memory, ps.Command)
	}
}

func processTop(app, id string) {
	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/processes/%s/top", app, id))

	if err != nil {
		stdcli.Error(err)
		return
	}

	var top ProcessTop

	err = json.Unmarshal(data, &top)

	if err != nil {
		stdcli.Error(err)
		return
	}

	longest := make([]int, len(top.Titles))

	for i, title := range top.Titles {
		longest[i] = len(title)
	}

	for _, process := range top.Processes {
		for i, data := range process {
			if len(data) > longest[i] {
				longest[i] = len(data)
			}
		}
	}

	fparts := make([]string, len(top.Titles))

	for i, l := range longest {
		fparts[i] = fmt.Sprintf("%%-%ds", l)
	}

	fp := strings.Join(fparts, " ") + "\n"

	fmt.Printf(fp, interfaceStrings(top.Titles)...)

	for _, p := range top.Processes {
		fmt.Printf(fp, interfaceStrings(p)...)
	}
}

func interfaceStrings(list []string) []interface{} {
	ret := make([]interface{}, len(list))

	for i, l := range list {
		ret[i] = l
	}

	return ret
}
