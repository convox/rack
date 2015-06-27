package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

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
	tags := m.TagNames("convox-charlie-935967921.us-east-1.elb.amazonaws.com:5000", proj, "123")

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
}
