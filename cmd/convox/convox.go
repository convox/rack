package main

import "github.com/convox/rack/cmd/convox/stdcli"

func init() {
	stdcli.DefaultWriter.Tags["app"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["release"] = stdcli.RenderAttributes(39)
	stdcli.DefaultWriter.Tags["security"] = stdcli.RenderAttributes(160)
	stdcli.DefaultWriter.Tags["warning"] = stdcli.RenderAttributes(220)
	stdcli.DefaultWriter.Tags["success"] = stdcli.RenderAttributes(10)
}
