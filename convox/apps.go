package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type App struct {
	Name string

	Status     string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Apps []App

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "apps",
		Action:      cmdApps,
		Description: "list deployed apps",
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new application",
				Usage:       "[name]",
				Action:      cmdAppCreate,
			},
			{
				Name:        "delete",
				Description: "delete an application",
				Usage:       "[name]",
				Action:      cmdAppDelete,
			},
			{
				Name:        "info",
				Description: "see info about an app",
				Usage:       "",
				Action:      cmdAppInfo,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdApps(c *cli.Context) {
	apps := getApps()

	longest := 3

	for _, app := range *apps {
		if len(app.Name) > longest {
			longest = len(app.Name)
		}
	}

	t := stdcli.NewTable("APP", "STATUS")

	for _, app := range *apps {
		t.AddRow(app.Name, app.Status)
	}

	t.Print()
}

func cmdAppCreate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) > 0 {
		app = c.Args()[0]
	}

	if app == "" {
		fmt.Printf("Creating app... ")
	} else {
		fmt.Printf("Creating app %s... ", app)
	}

	v := url.Values{}
	v.Set("name", app)
	data, err := ConvoxPostForm("/apps", v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	data, err = ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// poll for complete
	for {
		data, err = ConvoxGet(fmt.Sprintf("/apps/%s/status", app))

		if err != nil {
			stdcli.Error(err)
			return
		}

		if string(data) == "running" {
			break
		}

		time.Sleep(3 * time.Second)
	}

	if app == "" {
		fmt.Printf("OK, %s\n", a.Name)
	} else {
		fmt.Println("OK")
	}
}

func cmdAppDelete(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return
	}

	app := c.Args()[0]

	fmt.Printf("Deleting %s... ", app)

	_, err := ConvoxDelete(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("DELETING")
}

func cmdAppInfo(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	data, err := ConvoxGet("/apps/" + app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	var a *App
	err = json.Unmarshal(data, &a)

	if err != nil {
		stdcli.Error(err)
		return
	}

	matcher := regexp.MustCompile(`^(\w+)Port\d+Balancer`)

	ports := []string{}

	for key, value := range a.Outputs {
		if m := matcher.FindStringSubmatch(key); m != nil {
			ports = append(ports, fmt.Sprintf("%s:%s", strings.ToLower(m[1]), value))
		}
	}

	processes := []string{}

	for key, _ := range a.Parameters {
		if strings.HasSuffix(key, "Image") {
			processes = append(processes, strings.ToLower(key[0:len(key)-5]))
		}
	}

	sort.Strings(processes)

	if len(processes) == 0 {
		processes = append(processes, "(none)")
	}

	if len(ports) == 0 {
		ports = append(ports, "(none)")
	}

	release := a.Parameters["Release"]

	if release == "" {
		release = "(none)"
	}

	fmt.Printf("Name       %s\n", a.Name)
	fmt.Printf("Status     %s\n", a.Status)
	fmt.Printf("Release    %s\n", release)
	fmt.Printf("Processes  %s\n", strings.Join(processes, " "))

	if a.Outputs["BalancerHost"] != "" {
		fmt.Printf("Hostname   %s\n", a.Outputs["BalancerHost"])
		fmt.Printf("Ports      %s\n", strings.Join(ports, " "))
	}
}

func getApps() *Apps {
	data, err := ConvoxGet("/apps")

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	var apps *Apps
	err = json.Unmarshal(data, &apps)

	if err != nil {
		stdcli.Error(err)
		return nil
	}

	return apps
}
