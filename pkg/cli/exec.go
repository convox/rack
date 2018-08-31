package cli

import (
	"os"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("exec", "execute a command in a running process", Exec, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<pid> <command>",
		Validate: stdcli.ArgsMin(2),
	})
}

func Exec(rack sdk.Interface, c *stdcli.Context) error {
	pid := c.Arg(0)
	command := strings.Join(c.Args[1:], " ")

	opts := structs.ProcessExecOptions{}

	if w, h, err := c.TerminalSize(); err == nil {
		opts.Height = options.Int(h)
		opts.Width = options.Int(w)
	}

	if !stdcli.IsTerminal(os.Stdin) {
		opts.Tty = options.Bool(false)
	}

	restore := c.TerminalRaw()
	defer restore()

	code, err := rack.ProcessExec(app(c), pid, command, c, opts)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)
}
