package main

import (
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("install", "install a rack", RackInstall, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemInstallOptions{})),
		Usage:    "<type> [Parameter=Value]...",
		Validate: stdcli.ArgsMin(1),
	})
}
