package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/convox/build"
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

	Build(dir, app)

	m, err := build.ManifestFromPath(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		stdcli.Error(err)
		return
	}

	host, _, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return
	}

	host = strings.Split(host, ":")[0] + ":5000"

	if os.Getenv("REGISTRY_HOST") != "" {
		host = os.Getenv("REGISTRY_HOST")
	}

	prefix := strings.Replace(app, "-", "", -1)
	tag := fmt.Sprintf("%v", stdcli.Tagger())
	tags := m.Tags(host, prefix, tag)

	for tag, image := range tags {
		fmt.Printf("Tagging %s\n", image)
		err = stdcli.Run("docker", "tag", "-f", image, tag)

		if err != nil {
			stdcli.Error(err)
			return
		}

		fmt.Printf("Pushing %s\n", tag)
		err = stdcli.Run("docker", "push", tag)

		if err != nil {
			stdcli.Error(err)
			return
		}
	}

	// create app if it doesn't exist
	data, err := ConvoxGet(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		v := url.Values{}
		v.Set("name", app)
		data, err = ConvoxPostForm("/apps", v)

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

			time.Sleep(1000 * time.Millisecond)
		}
	}

	// create release
	v := url.Values{}
	v.Set("manifest", m.String())
	v.Set("tag", tag)
	data, err = ConvoxPostForm(fmt.Sprintf("/apps/%s/releases", app), v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Releasing %s\n", tag)

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

		time.Sleep(1000 * time.Millisecond)
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

	a.PrintInfo()
}
