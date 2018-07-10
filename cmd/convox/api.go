package main

import (
	"encoding/json"

	"github.com/convox/stdcli"
	"github.com/convox/stdsdk"
)

func init() {
	CLI.Command("api get", "query the rack api", Api, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<path>",
		Validate: stdcli.Args(1),
	})
}

func Api(c *stdcli.Context) error {
	var v interface{}

	if err := provider(c).Get(c.Arg(0), stdsdk.RequestOptions{}, &v); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return c.Writef("%s\n", string(data))
}
