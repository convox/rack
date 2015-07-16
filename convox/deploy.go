package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "deploy",
		Description: "deploy an app to AWS",
		Usage:       "<directory>",
		Action:      cmdDeploy,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdDeploy(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// create app if it doesn't exist
	data, err := ConvoxGet(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		v := url.Values{}
		v.Set("name", app)
		data, err = ConvoxPostForm("/apps", v)

		if stdcli.Debug() {
			fmt.Fprintf(os.Stderr, "DEBUG: POST /apps response: '%v', '%v'\n", string(data), err)
		}

		if err != nil {
			stdcli.Error(err)
			return
		}

		fmt.Printf("Created app %s\n", app)

		// poll for complete
		for {
			data, err = ConvoxGet(fmt.Sprintf("/apps/%s/status", app))

			if err != nil {
				stdcli.Error(err)
				return
			}

			if string(data) == "running" {
				fmt.Printf("Status %s\n", data)
				break
			}

			if stdcli.Debug() {
				fmt.Fprintf(os.Stderr, "DEBUG: POST /apps response: '%v', '%v'\n", string(data), err)
			}

			time.Sleep(1000 * time.Millisecond)
		}
	}

	// build
	release, err := executeBuild(dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Print("Releasing... ")

	// promote release
	data, err = ConvoxPost(fmt.Sprintf("/apps/%s/releases/%s/promote", app, release), "")

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

		time.Sleep(1 * time.Second)
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

	fmt.Printf("OK, %s\n", a.Parameters["Release"])

	matcher := regexp.MustCompile(`^(\w+)Port\d+Balancer`)

	if host, ok := a.Outputs["BalancerHost"]; ok {
		for key, value := range a.Outputs {
			if m := matcher.FindStringSubmatch(key); m != nil {
				fmt.Printf("%s: http://%s:%s\n", strings.ToLower(m[1]), host, value)
			}
		}
	}
}
