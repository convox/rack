package cli

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
	cv "github.com/convox/version"
	update "github.com/inconshreveable/go-update"
)

func init() {
	registerWithoutProvider("update", "update the cli", Update, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.ArgsMax(1),
	})
}

func Update(rack sdk.Interface, c *stdcli.Context) error {
	target := c.Arg(0)

	// if no version specified, find the latest version
	if target == "" {
		v, err := cv.Latest()
		if err != nil {
			return err
		}

		target = v
	}

	url := fmt.Sprintf("https://s3.amazonaws.com/convox/release/%s/cli/%s/%s", target, runtime.GOOS, executableName())

	res, err := http.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("invalid version")
	}

	defer res.Body.Close()

	c.Startf("Updating to <release>%s</release>", target)

	if err := update.Apply(res.Body, update.Options{}); err != nil {
		return err
	}

	return c.OK()
}
