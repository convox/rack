package cli

import (
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

type Engine struct {
	*stdcli.Engine
	Client sdk.Interface
}

func (e *Engine) Command(command, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	wfn := func(c *stdcli.Context) error {
		return fn(e.currentClient(c), c)
	}

	e.Engine.Command(command, description, wfn, opts)
}

func (e *Engine) CommandWithoutProvider(command, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	wfn := func(c *stdcli.Context) error {
		return fn(nil, c)
	}

	e.Engine.Command(command, description, wfn, opts)
}

func (e *Engine) RegisterCommands() {
	for _, c := range commands {
		if c.Rack {
			e.Command(c.Command, c.Description, c.Handler, c.Opts)
		} else {
			e.CommandWithoutProvider(c.Command, c.Description, c.Handler, c.Opts)
		}
	}
}

var commands = []command{}

type command struct {
	Command     string
	Description string
	Handler     HandlerFunc
	Opts        stdcli.CommandOptions
	Rack        bool
}

func register(cmd, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	commands = append(commands, command{
		Command:     cmd,
		Description: description,
		Handler:     fn,
		Opts:        opts,
		Rack:        true,
	})
}

func registerWithoutProvider(cmd, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	commands = append(commands, command{
		Command:     cmd,
		Description: description,
		Handler:     fn,
		Opts:        opts,
		Rack:        false,
	})
}
