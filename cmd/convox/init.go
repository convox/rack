package main

import (
	"fmt"
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

	err = initApplication(dir)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("cli-init", distinctId, ep)
	return nil
}

func initApplication(dir string) error {
	// TODO parse the Dockerfile and build a docker-compose.yml
	if helpers.Exists("Dockerfile") || helpers.Exists("docker-compose.yml") {
		return nil
	}

	var fw appify.Framework

	kind := helpers.DetectApplication(dir)

	switch kind {
	case "heroku":
		fw = &appify.Buildpack{}

	default:
		ga := &appify.GenericApp{}
		ga.AppKind = kind
		fw = ga
	}

	fmt.Printf("Initializing a %s app\n", kind)
	if err := fw.Setup(dir); err != nil {
		return err
	}

	return fw.Appify()
}
