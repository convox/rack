package main

import (
	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "scale",
		Description: "scale an app's processes",
		Usage:       "PROCESS [--count 2] [--memory 512]",
		Action:      cmdScale,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
			cli.IntFlag{
				Name:  "count",
				Value: 1,
				Usage: "Number of processes to keep running for specified process type.",
			},
			cli.IntFlag{
				Name:  "memory",
				Value: 256,
				Usage: "Amount of memory, in MB, available to specified process type.",
			},
		},
	})
}

func cmdScale(c *cli.Context) {
	// _, app, err := stdcli.DirApp(c, ".")

	// if err != nil {
	//   stdcli.Error(err)
	//   return
	// }

	// v := url.Values{}

	// if c.IsSet("count") {
	//   v.Set("count", c.String("count"))
	// }

	// if c.IsSet("memory") {
	//   v.Set("mem", c.String("memory"))
	// }

	// if len(v) > 0 {
	//   v.Set("process", c.Args()[0])

	//   _, err = ConvoxPostForm("/apps/"+app, v)

	//   if err != nil {
	//     stdcli.Error(err)
	//     return
	//   }
	// }

	// data, err := ConvoxGet("/apps/" + app)

	// if err != nil {
	//   stdcli.Error(err)
	//   return
	// }

	// var a *App
	// err = json.Unmarshal(data, &a)

	// if err != nil {
	//   stdcli.Error(err)
	//   return
	// }

	// processes := map[string]Process{}
	// names := []string{}

	// for k, v := range a.Parameters {
	//   if !strings.HasSuffix(k, "DesiredCount") {
	//     continue
	//   }

	//   ps := strings.Replace(k, "DesiredCount", "", 1)
	//   p := strings.ToLower(ps)

	//   i, err := strconv.ParseInt(v, 10, 64)

	//   if err != nil {
	//     stdcli.Error(err)
	//     return
	//   }

	//   m, err := strconv.ParseInt(a.Parameters[ps+"Memory"], 10, 64)

	//   if err != nil {
	//     stdcli.Error(err)
	//     return
	//   }

	//   processes[p] = Process{Name: p, Count: i, Memory: m}
	//   names = append(names, p)
	// }

	// sort.Strings(names)

	// t := stdcli.NewTable("PROCESS", "COUNT", "MEM")

	// for _, name := range names {
	//   ps := processes[name]
	//   t.AddRow(ps.Name, fmt.Sprintf("%d", ps.Count), fmt.Sprintf("%d", ps.Memory))
	// }

	// t.Print()
}
