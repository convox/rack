package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/convox/stdcli"
	cv "github.com/convox/version"
	update "github.com/inconshreveable/go-update"
)

func init() {
	CLI.Command("update", "update the cli", Update, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.ArgsMax(1),
	})
}

func Update(c *stdcli.Context) error {
	target, err := cv.Latest()
	if err != nil {
		return err
	}

	fmt.Printf("target = %+v\n", target)

	url := fmt.Sprintf("https://s3.amazonaws.com/convox/release/%s/cli/%s/%s", target, runtime.GOOS, executableName())

	fmt.Printf("url = %+v\n", url)

	res, err := http.Get(url)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	c.Startf("Updating to <release>%s</release>", target)

	if err := update.Apply(res.Body, update.Options{}); err != nil {
		return err
	}

	return c.OK()
}
