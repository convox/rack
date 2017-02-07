package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/app"
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

	if helpers.Exists("docker-compose.yml") {
		fmt.Println("docker-compose.yml already exists; try running convox start or")
		fmt.Println(nextStepsText["unknown"])
		return nil
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
	var fw app.Framework

	kind := helpers.DetectApplication(dir)

	switch {
	case strings.Contains(kind, "heroku"):
		fw = &app.Buildpack{
			Kind: strings.Split(kind, "/")[1],
		}

	default:
		ga := &app.GenericApp{
			Kind: kind,
		}
		fw = ga
	}

	fmt.Printf("Initializing a %s app\n", kind)
	if err := fw.Setup(dir); err != nil {
		return kind, err
	}

	err := fw.Appify()

	if strings.Contains(kind, "heroku") {
		fmt.Println(nextStepsText["heroku"])
	}

	if val, ok := nextStepsText[kind]; ok {
		fmt.Println(val)
	}

	return kind, err
}

var nextStepsText = map[string]string{
	"django":  "Try `convox start`. See https://convox.com/docs/django/ for more information.",
	"heroku":  "Try `convox start`. See https://convox.com/guide/heroku/ for more information.",
	"rails":   "Try `convox start`. See https://convox.com/docs/rails/ for more information.",
	"sinatra": "Try `convox start`. See https://convox.com/docs/sinatra/ for more information.",
	"unknown": "See https://convox.com/docs/preparing-an-application/ for more information on preparing an app.",
}
