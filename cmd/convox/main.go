package main

import (
	"os"

	"github.com/convox/rack/pkg/cli"
	"github.com/convox/stdcli"
)

var (
	version = "dev"
)

func main() {
	c := cli.New("convox", version)

	c.Writer.Tags["app"] = stdcli.RenderColors(39)
	c.Writer.Tags["build"] = stdcli.RenderColors(23)
	c.Writer.Tags["rack"] = stdcli.RenderColors(26)
	c.Writer.Tags["process"] = stdcli.RenderColors(27)
	c.Writer.Tags["release"] = stdcli.RenderColors(24)
	c.Writer.Tags["service"] = stdcli.RenderColors(25)

	if dir := os.Getenv("CONVOX_CONFIG"); dir != "" {
		c.Settings = dir
	}

	os.Exit(c.Execute(os.Args[1:]))
}
