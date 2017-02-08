package main

import (
	"fmt"
	"os/exec"
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
		fmt.Println("docker-compose.yml already exists; try running convox start instead")
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

	out, err := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:/tmp/app", dir), "convox/init", "detect").CombinedOutput()
	if err != nil {
		return "", err
	}
	kind := strings.TrimSpace(string(out))

	fw = &app.Buildpack{
		Kind: kind,
	}

	fmt.Printf("Initializing a %s app\n", kind)
	if err := fw.Setup(dir); err != nil {
		return kind, err
	}

	err = fw.Appify()
	fmt.Println("For more information on preparing an app check out https://convox.com/docs/preparing-an-application/")

	return kind, err
}

func detectKind(kind string) string {

	switch kind {
	case "Node.js":
		return "nodejs"
	}

	return kind // this shoudn't be reached
}
