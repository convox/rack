package main

import (
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
)

var dockerBin = helpers.DetectDocker()

func init() {
	stdcli.DefaultWriter.Tags["app"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["dir"] = stdcli.RenderAttributes(246)
	stdcli.DefaultWriter.Tags["fail"] = stdcli.RenderAttributes(160)
	stdcli.DefaultWriter.Tags["release"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["service"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["setting"] = stdcli.RenderAttributes(246)
}
