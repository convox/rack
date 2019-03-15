package cli

import (
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("racks", "list available racks", Racks, stdcli.CommandOptions{
		Validate: stdcli.Args(0),
	})
}

func Racks(rack sdk.Interface, c *stdcli.Context) error {
	rs, err := racks(c)
	if err != nil {
		return err
	}

	t := c.Table("NAME", "STATUS")

	for _, r := range rs {
		t.AddRow(r.Name, r.Status)
	}

	return t.Print()
}
