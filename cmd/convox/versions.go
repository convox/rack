package main

import (
	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/release/version"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "versions",
		Description: "list convox versions",
		Usage:       "",
		Action:      cmdVersions,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "unpublished",
				Usage: "include unpublished versions",
			},
		},
	})
}

func cmdVersions(c *cli.Context) {
	vs, err := version.All()

	if err != nil {
		return
	}

	var t *stdcli.Table

	if c.Bool("unpublished") {
		t = stdcli.NewTable("RELEASE", "PUBLISHED", "REQUIRED", "DESCRIPTION")
		for _, v := range vs {
			published := "false"
			if v.Published {
				published = "true"
			}
			required := "false"
			if v.Required {
				required = "true"
			}
			t.AddRow(v.Version, published, required, v.Description)
		}
	} else {
		t = stdcli.NewTable("RELEASE", "REQUIRED", "DESCRIPTION")
		for _, v := range vs {
			if v.Published {
				required := "false"
				if v.Required {
					required = "true"
				}
				t.AddRow(v.Version, required, v.Description)
			}
		}
	}

	t.Print()
}
