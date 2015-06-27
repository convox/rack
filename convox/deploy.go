package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
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
	})
}

func cmdDeploy(c *cli.Context) {
	base := "."

	if len(c.Args()) > 0 {
		base = c.Args()[0]
	}

	base, err := filepath.Abs(base)

	if err != nil {
		stdcli.Error(err)
		return
	}

	proj := strings.Replace(filepath.Base(base), "-", "", -1)

	cmdBuild(c)

	dat, err := ioutil.ReadFile(filepath.Join(base, "docker-compose.yml"))

	if err != nil {
		stdcli.Error(err)
		return
	}

	m, _ := build.ManifestFromBytes(dat)

	images := m.ImageNames(proj)
	tag := "123"
	tags := m.TagNames("convox-charlie-935967921.us-east-1.elb.amazonaws.com:5000", proj, tag)

	for i := 0; i < len(images); i++ {
		fmt.Printf("tag %s %s\n", images[i], tags[i])
		err = stdcli.Run("docker", "tag", "-f", images[i], tags[i])

		if err != nil {
			stdcli.Error(err)
			return
		}

		err = stdcli.Run("docker", "push", tags[i])

		if err != nil {
			stdcli.Error(err)
			return
		}
	}

	// create app
	v := url.Values{}
	v.Set("name", proj)
	data, err := ConvoxPostForm("/apps", v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Created app %s\n", proj)

	// poll for complete
	for {
		data, err = ConvoxGet(fmt.Sprintf("/apps/%s/status", proj))

		if err != nil {
			stdcli.Error(err)
			return
		}

		if string(data) == "complete" {
			fmt.Printf("Status %s\n", data)
			break
		}

		time.Sleep(1000 * time.Millisecond)
	}

	// create release
	v = url.Values{}
	v.Set("manifest", m.String())
	v.Set("tag", tag)
	data, err = ConvoxPostForm(fmt.Sprintf("/apps/%s/releases", proj), v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Created release %s\n", tag)

	// poll for complete
	for {
		data, err = ConvoxGet(fmt.Sprintf("/apps/%s/status", proj))

		if err != nil {
			stdcli.Error(err)
			return
		}

		if string(data) == "complete" {
			fmt.Printf("Status %s\n", data)
			break
		}

		time.Sleep(1000 * time.Millisecond)
	}
}
