package cli

import (
	"encoding/json"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
	"github.com/convox/stdsdk"
)

func init() {
	register("api get", "query the rack api", Api, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<path>",
		Validate: stdcli.Args(1),
	})
}

func Api(rack sdk.Interface, c *stdcli.Context) error {
	var v interface{}

	if err := rack.Get(c.Arg(0), stdsdk.RequestOptions{}, &v); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return c.Writef("%s\n", string(data))
}
