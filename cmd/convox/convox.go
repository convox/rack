package main

import (
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
)

var dockerBin = helpers.DetectDocker()

func init() {
	stdcli.DefaultWriter.Tags["app"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["config"] = stdcli.RenderAttributes()
	stdcli.DefaultWriter.Tags["description"] = stdcli.RenderAttributes(245)
	stdcli.DefaultWriter.Tags["fail"] = stdcli.RenderAttributes(160)
	stdcli.DefaultWriter.Tags["file"] = stdcli.RenderAttributes(249)
	stdcli.DefaultWriter.Tags["link"] = stdcli.RenderAttributes(4)
	stdcli.DefaultWriter.Tags["linenumber"] = stdcli.RenderAttributes(235)
	stdcli.DefaultWriter.Tags["release"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["security"] = stdcli.RenderAttributes(226)
	stdcli.DefaultWriter.Tags["service"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["database"] = stdcli.RenderAttributes(141)
	stdcli.DefaultWriter.Tags["success"] = stdcli.RenderAttributes(10)
	stdcli.DefaultWriter.Tags["unsupported"] = stdcli.RenderAttributes(220)
	stdcli.DefaultWriter.Tags["warning"] = stdcli.RenderAttributes(172)
}
