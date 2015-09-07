package main

import (
	"fmt"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

// type GetMetricStatisticsOutput struct {
//   Datapoints []*Datapoint `type:"list"`
//   Label      *string      `type:"string"`
// }

// type ProcessTop struct {
//   Titles    []string
//   Processes [][]string
// }

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ps",
		Description: "list an app's processes",
		Usage:       "",
		Action:      cmdPs,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
		// {
		//   Name:        "stop",
		//   Description: "stop a process",
		//   Usage:       "<id>",
		//   Action:      cmdPsStop,
		//   Flags:       []cli.Flag{appFlag},
		// },
		// {
		//   Name:        "top",
		//   Description: "view utilization stats for a given process type",
		//   Usage:       "<process>",
		//   Action:      cmdPsTop,
		//   Flags:       []cli.Flag{appFlag},
		// },
		},
	})
}

func cmdPs(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	ps, err := rackClient().GetProcesses(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("NAME", "COUNT", "MEMORY", "PORTS", "COMMAND")

	for _, p := range ps {
		ports := make([]string, len(p.Ports))

		for i, pp := range p.Ports {
			ports[i] = fmt.Sprintf("%d", pp)
		}

		t.AddRow(p.Name, fmt.Sprintf("%d", p.Count), fmt.Sprintf("%d", p.Memory), strings.Join(ports, " "), p.Command)
	}

	t.Print()
}

// func cmdPsStop(c *cli.Context) {
//   _, app, err := stdcli.DirApp(c, ".")

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   if len(c.Args()) != 1 {
//     stdcli.Usage(c, "stop")
//     return
//   }

//   id := c.Args()[0]

//   _, err = ConvoxDelete(fmt.Sprintf("/apps/%s/processes/%s", app, id))

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   fmt.Printf("Stopping %s\n", id)
// }

// func cmdPsTop(c *cli.Context) {
//   _, app, err := stdcli.DirApp(c, ".")

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   if len(c.Args()) != 1 {
//     stdcli.Usage(c, "top")
//     return
//   }

//   process := c.Args()[0]

//   data, err := ConvoxGet(fmt.Sprintf("/apps/%s/process_types/%s/top", app, process))

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   if string(data) == "null" {
//     stdcli.Error(fmt.Errorf("No process named %s", process))
//     return
//   }

//   var outputs []GetMetricStatisticsOutput

//   err = json.Unmarshal(data, &outputs)

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   t := stdcli.NewTable("", "MIN", "AVG", "MAX", "UPDATED")
//   label := ""

//   for _, output := range outputs {
//     switch *output.Label {
//     case "MemoryUtilization":
//       label = "MEM"
//     case "CPUUtilization":
//       label = "CPU"
//     }

//     if len(output.Datapoints) == 0 {
//       stdcli.Error(fmt.Errorf("No %s data available", process))
//       return
//     }

//     dp := output.Datapoints[0]

//     t.AddRow(label, fmt.Sprintf("%.1f%%", dp.Minimum), fmt.Sprintf("%.1f%%", dp.Average), fmt.Sprintf("%.1f%%", dp.Maximum), humanize.Time(dp.Timestamp))
//   }

//   t.Print()
// }

// func processTop(app, id string) {
//   data, err := ConvoxGet(fmt.Sprintf("/apps/%s/processes/%s/top", app, id))

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   var top ProcessTop

//   err = json.Unmarshal(data, &top)

//   if err != nil {
//     stdcli.Error(err)
//     return
//   }

//   longest := make([]int, len(top.Titles))

//   for i, title := range top.Titles {
//     longest[i] = len(title)
//   }

//   for _, process := range top.Processes {
//     for i, data := range process {
//       if len(data) > longest[i] {
//         longest[i] = len(data)
//       }
//     }
//   }

//   fparts := make([]string, len(top.Titles))

//   for i, l := range longest {
//     fparts[i] = fmt.Sprintf("%%-%ds", l)
//   }

//   fp := strings.Join(fparts, " ") + "\n"

//   fmt.Printf(fp, interfaceStrings(top.Titles)...)

//   for _, p := range top.Processes {
//     fmt.Printf(fp, interfaceStrings(p)...)
//   }
// }
