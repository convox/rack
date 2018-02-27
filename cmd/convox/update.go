package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/convox/rack/cmd/convox/stdcli"
	update "github.com/inconshreveable/go-update"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "update",
		Description: "update the cli",
		Usage:       "[version]",
		Action:      cmdUpdate,
		Flags:       []cli.Flag{rackFlag},
	})
}

func cmdUpdate(c *cli.Context) error {
	version, err := latestVersion()
	if err != nil {
		return err
	}

	if len(c.Args()) > 0 {
		version = c.Args()[0]
	}

	stdcli.Spinner.Prefix = fmt.Sprintf("Updating convox to %s: ", version)
	stdcli.Spinner.Start()

	exe := "convox"

	if runtime.GOOS == "windows" {
		exe = "convox.exe"
	}

	url := fmt.Sprintf("https://s3.amazonaws.com/convox/release/%s/cli/%s/%s", version, runtime.GOOS, exe)

	res, err := http.Get(url)
	if err != nil {
		return stdcli.Error(err)
	}

	defer res.Body.Close()

	if err := update.Apply(res.Body, update.Options{}); err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("\x08\x08OK\n")

	stdcli.Spinner.Stop()

	return nil
}
