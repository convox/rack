package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/pkg/start"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

type HandlerFunc func(sdk.Interface, *stdcli.Context) error

var (
	Starter      = start.New()
	WaitDuration = 5 * time.Second
)

var (
	flagApp      = stdcli.StringFlag("app", "a", "app name")
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
	e.Writer.Tags["dir"] = stdcli.RenderColors(246)
	e.Writer.Tags["build"] = stdcli.RenderColors(23)
	e.Writer.Tags["fail"] = stdcli.RenderColors(160)
	e.Writer.Tags["rack"] = stdcli.RenderColors(26)
	e.Writer.Tags["process"] = stdcli.RenderColors(27)
	e.Writer.Tags["release"] = stdcli.RenderColors(24)
	e.Writer.Tags["service"] = stdcli.RenderColors(25)
	e.Writer.Tags["setting"] = stdcli.RenderColors(246)
	e.Writer.Tags["system"] = stdcli.RenderColors(15)

	for i := 0; i < 18; i++ {
		e.Writer.Tags[fmt.Sprintf("color%d", i)] = stdcli.RenderColors(237 + i)
	}

	if dir := os.Getenv("CONVOX_CONFIG"); dir != "" {
		e.Settings = dir
	}

	e.RegisterCommands()

	return e
}
