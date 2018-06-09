package main

import (
	"os"

	"github.com/convox/stdcli"
)

var (
	version = "dev"
)

var (
	CLI = stdcli.New("convox", version)
)

var (
	flagApp  = stdcli.StringFlag("app", "a", "app name")
	flagId   = stdcli.BoolFlag("id", "", "put logs on stderr, release id on stdout")
	flagRack = stdcli.StringFlag("rack", "r", "rack name")
	flagWait = stdcli.BoolFlag("wait", "w", "wait for completion")
)

func init() {
	CLI.Writer.Tags["app"] = stdcli.RenderColors(39)
	CLI.Writer.Tags["build"] = stdcli.RenderColors(23)
	CLI.Writer.Tags["rack"] = stdcli.RenderColors(26)
	CLI.Writer.Tags["release"] = stdcli.RenderColors(24)
	CLI.Writer.Tags["service"] = stdcli.RenderColors(25)
}

func main() {
	os.Exit(CLI.Execute(os.Args[1:]))
}
