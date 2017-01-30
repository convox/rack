package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/appify"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "init",
		Description: "initialize an app for local development",
		Usage:       "[directory]",
		Action:      cmdInit,
	})
}

func cmdInit(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	// TODO parse the Dockerfile and build a docker-compose.yml
	if helpers.Exists("docker-compose.yml") {
		return stdcli.Error(fmt.Errorf("Cannot initialize a project that already contains docker-compose.yml"))

	}

	appType, err := initApplication(dir, distinctId)
	if err != nil {
		stdcli.QOSEventSend("Dev Code Update Failed", distinctId, stdcli.QOSEventProperties{Error: err, AppType: appType})
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err, AppType: appType})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("Dev Code Updated", distinctId, stdcli.QOSEventProperties{AppType: appType})
	stdcli.QOSEventSend("cli-init", distinctId, ep)
	return nil
}

func initApplication(dir, distinctId string) (string, error) {
	// TODO parse the Dockerfile and build a docker-compose.yml
	if helpers.Exists("Dockerfile") || helpers.Exists("docker-compose.yml") {
		return "docker", nil
	}

	var fw appify.Framework

	kind := helpers.DetectApplication(dir)

	switch {
	case strings.Contains(kind, "heroku"):
		fw = &appify.Buildpack{
			Kind: strings.Split(kind, "/")[1],
		}

	default:
		ga := &appify.GenericApp{
			Kind: kind,
		}
		fw = ga
	}

	fmt.Printf("Initializing a %s app\n", kind)
	if err := fw.Setup(dir); err != nil {
		return kind, err
	}

	err := fw.Appify()
	return kind, err
}
