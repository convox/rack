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

	distinctID, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err})
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	files := []string{
		"Dockerfile",
		"docker-compose.yml",
	}
	for _, file := range files {
		// TODO When only a Dockerfile exists, parse it and build a docker-compose.yml
		if helpers.Exists(file) {
			e := fmt.Errorf("Cannot initialize an app that already contains a %s", file)
			stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{ValidationError: e})
			return stdcli.Error(e)
		}
	}

	appType, err := initApplication(dir)
	if err != nil {
		stdcli.QOSEventSend("Dev Code Update Failed", distinctID, stdcli.QOSEventProperties{Error: err, AppType: appType})
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err, AppType: appType})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("Dev Code Updated", distinctID, stdcli.QOSEventProperties{AppType: appType})
	stdcli.QOSEventSend("cli-init", distinctID, ep)
	return nil
}

func initApplication(dir string) (string, error) {
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
