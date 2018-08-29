package cli

import (
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

type HandlerFunc func(sdk.Interface, *stdcli.Context) error

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

	e.RegisterCommands()

	return e
}
