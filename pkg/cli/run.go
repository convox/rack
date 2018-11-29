package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("run", "execute a command in a new process", Run, stdcli.CommandOptions{
		Flags: append(
			stdcli.OptionFlags(structs.ProcessRunOptions{}),
			flagRack,
			flagApp,
			stdcli.BoolFlag("detach", "d", "run process in the background"),
			stdcli.IntFlag("timeout", "t", "timeout"),
		),
		Usage:    "<service> <command>",
		Validate: stdcli.ArgsMin(2),
	})
}

func Run(rack sdk.Interface, c *stdcli.Context) error {
	// s, err := rack.SystemGet()
	// if err != nil {
	//   return err
	// }

	service := c.Arg(0)
	command := strings.Join(c.Args[1:], " ")

	var opts structs.ProcessRunOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	opts.Command = options.String(command)

	if c.Bool("detach") {
		c.Startf("Running detached process")

		if err := runDetached(rack, app(c), service, command); err != nil {
			return err
		}
	}

	timeout := 3600

	if t := c.Int("timeout"); t > 0 {
		timeout = t
	}

	if w, h, err := c.TerminalSize(); err == nil {
		opts.Height = options.Int(h)
		opts.Width = options.Int(w)
	}

	restore := c.TerminalRaw()
	defer restore()

	code, err := runAttached(c, rack, app(c), service, opts, timeout)
	if err != nil {
		return err
	}

	return stdcli.Exit(code)

	// if s.Version <= "20180708231844" {
	//   if c.Bool("detach") {
	//     c.Startf("Running detached process")

	//     pid, err := rack.ProcessRunDetached(app(c), service, opts)
	//     if err != nil {
	//       return err
	//     }

	//     return c.OK(pid)
	//   }

	//   code, err := rack.ProcessRunAttached(app(c), service, c, timeout, opts)
	//   if err != nil {
	//     return err
	//   }

	//   return stdcli.Exit(code)
	// }

	// if c.Bool("detach") {
	//   c.Startf("Running detached process")

	//   ps, err := rack.ProcessRun(app(c), service, opts)
	//   if err != nil {
	//     return err
	//   }

	//   return c.OK(ps.Id)
	// }

	// opts.Command = options.String(fmt.Sprintf("sleep %d", timeout))

	// ps, err := rack.ProcessRun(app(c), c.Arg(0), opts)
	// if err != nil {
	//   return err
	// }

	// defer rack.ProcessStop(app(c), ps.Id)

	// if err := waitForProcessRunning(rack, c, app(c), ps.Id); err != nil {
	//   return err
	// }

	// eopts := structs.ProcessExecOptions{
	//   Entrypoint: options.Bool(true),
	//   Height:     opts.Height,
	//   Width:      opts.Width,
	// }

	// if !stdcli.IsTerminal(os.Stdin) {
	//   eopts.Tty = options.Bool(false)
	// }

	// code, err := rack.ProcessExec(app(c), ps.Id, command, c, eopts)
	// if err != nil {
	//   return err
	// }

	// return stdcli.Exit(code)
}

func runAttached(rw io.ReadWriter, rack sdk.Interface, app, service string, opts structs.ProcessRunOptions, timeout int) (int, error) {
	s, err := rack.SystemGet()
	if err != nil {
		return 0, err
	}

	if s.Version <= "20180708231844" {
		return rack.ProcessRunAttached(app, service, rw, timeout, opts)
	}

	command := helpers.DefaultString(opts.Command, "")

	opts.Command = options.String(fmt.Sprintf("sleep %d", timeout))

	ps, err := rack.ProcessRun(app, service, opts)
	if err != nil {
		return 0, err
	}

	defer rack.ProcessStop(app, ps.Id)

	if err := waitForProcessRunning(rack, app, ps.Id); err != nil {
		return 0, err
	}

	eopts := structs.ProcessExecOptions{
		Entrypoint: options.Bool(true),
		Height:     opts.Height,
		Width:      opts.Width,
	}

	if !stdcli.IsTerminal(os.Stdin) {
		eopts.Tty = options.Bool(false)
	}

	return rack.ProcessExec(app, ps.Id, command, rw, eopts)
}

func runDetached(rack sdk.Interface, app, service, command string) error {
	return nil
}
