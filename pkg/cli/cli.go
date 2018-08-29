package cli

import (
	"os"
	"time"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

type HandlerFunc func(sdk.Interface, *stdcli.Context) error

var (
	WaitDuration = 5 * time.Second
)

var (
	flagApp      = stdcli.StringFlag("app", "a", "app name")
	flagForce    = stdcli.BoolFlag("force", "", "proceed without confirmation")
	flagId       = stdcli.BoolFlag("id", "", "put logs on stderr, release id on stdout")
	flagNoFollow = stdcli.BoolFlag("no-follow", "", "do not follow logs")
	flagRack     = stdcli.StringFlag("rack", "r", "rack name")
	flagWait     = stdcli.BoolFlag("wait", "w", "wait for completion")
)

func New(name, version string) *Engine {
	e := &Engine{
		Engine: stdcli.New(name, version),
	}

	e.Writer.Tags["app"] = stdcli.RenderColors(39)
	e.Writer.Tags["build"] = stdcli.RenderColors(23)
	e.Writer.Tags["rack"] = stdcli.RenderColors(26)
	e.Writer.Tags["process"] = stdcli.RenderColors(27)
	e.Writer.Tags["release"] = stdcli.RenderColors(24)
	e.Writer.Tags["service"] = stdcli.RenderColors(25)

	if dir := os.Getenv("CONVOX_CONFIG"); dir != "" {
		e.Settings = dir
	}

	e.RegisterCommands()

	return e
}
