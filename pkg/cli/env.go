package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("env", "list env vars", Env, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Validate: stdcli.Args(0),
	})

	register("env edit", "edit env interactively", EnvEdit, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagRack,
			flagWait,
			stdcli.BoolFlag("promote", "p", "promote the release"),
		},
		Validate: stdcli.Args(0),
	})

	register("env get", "get an env var", EnvGet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<var>",
		Validate: stdcli.Args(1),
	})

	register("env set", "set env var(s)", EnvSet, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagId,
			flagRack,
			flagWait,
			stdcli.BoolFlag("replace", "", "replace all environment variables with given ones"),
			stdcli.BoolFlag("promote", "p", "promote the release"),
		},
		Usage: "<key=value> [key=value]...",
	})

	register("env unset", "unset env var(s)", EnvUnset, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagId,
			flagRack,
			flagWait,
			stdcli.BoolFlag("promote", "p", "promote the release"),
		},
		Usage:    "<key> [key]...",
		Validate: stdcli.ArgsMin(1),
	})
}

func Env(rack sdk.Interface, c *stdcli.Context) error {
	env, err := helpers.AppEnvironment(rack, app(c))
	if err != nil {
		return err
	}

	c.Writef("%s\n", env.String())

	return nil
}

func EnvEdit(rack sdk.Interface, c *stdcli.Context) error {
	env, err := helpers.AppEnvironment(rack, app(c))
	if err != nil {
		return err
	}

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	file := filepath.Join(tmp, fmt.Sprintf("%s.env", app(c)))

	fd, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	if _, err := fd.Write([]byte(env.String())); err != nil {
		return err
	}

	fd.Close()

	editor := "vi"

	if e := os.Getenv("EDITOR"); e != "" {
		editor = e
	}

	if err := c.Terminal(editor, file); err != nil {
		return err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	nenv := structs.Environment{}

	if err := nenv.Load(bytes.TrimSpace(data)); err != nil {
		return err
	}

	nks := []string{}

	for k := range nenv {
		nks = append(nks, fmt.Sprintf("<info>%s</info>", k))
	}

	sort.Strings(nks)

	c.Startf(fmt.Sprintf("Setting %s", strings.Join(nks, ", ")))

	var r *structs.Release

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		r, err = rack.EnvironmentSet(app(c), []byte(nenv.String()))
		if err != nil {
			return err
		}
	} else {
		r, err = rack.ReleaseCreate(app(c), structs.ReleaseCreateOptions{Env: options.String(nenv.String())})
		if err != nil {
			return err
		}
	}

	c.OK()

	c.Writef("Release: <release>%s</release>\n", r.Id)

	if c.Bool("promote") {
		if err := releasePromote(rack, c, app(c), r.Id); err != nil {
			return err
		}
	}

	return nil
}

func EnvGet(rack sdk.Interface, c *stdcli.Context) error {
	env, err := helpers.AppEnvironment(rack, app(c))
	if err != nil {
		return err
	}

	k := c.Arg(0)

	v, ok := env[k]
	if !ok {
		return fmt.Errorf("env not found: %s", k)
	}

	c.Writef("%s\n", v)

	return nil
}

func EnvSet(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	env := structs.Environment{}
	var err error

	if !c.Bool("replace") {
		env, err = helpers.AppEnvironment(rack, app(c))
		if err != nil {
			return err
		}
	}

	args := []string(c.Args)
	keys := []string{}

	if !c.Reader().IsTerminal() {
		s := bufio.NewScanner(c.Reader())
		for s.Scan() {
			args = append(args, s.Text())
		}
	}

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			keys = append(keys, fmt.Sprintf("<info>%s</info>", parts[0]))
			env[parts[0]] = parts[1]
		}
	}

	sort.Strings(keys)

	c.Startf(fmt.Sprintf("Setting %s", strings.Join(keys, ", ")))

	var r *structs.Release

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		r, err = rack.EnvironmentSet(app(c), []byte(env.String()))
		if err != nil {
			return err
		}
	} else {
		r, err = rack.ReleaseCreate(app(c), structs.ReleaseCreateOptions{Env: options.String(env.String())})
		if err != nil {
			return err
		}
	}

	c.OK()

	c.Writef("Release: <release>%s</release>\n", r.Id)

	if c.Bool("promote") {
		if err := releasePromote(rack, c, app(c), r.Id); err != nil {
			return err
		}
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, r.Id)
	}

	return nil
}

func EnvUnset(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	env, err := helpers.AppEnvironment(rack, app(c))
	if err != nil {
		return err
	}

	keys := []string{}

	for _, arg := range c.Args {
		keys = append(keys, fmt.Sprintf("<info>%s</info>", arg))
		delete(env, arg)
	}

	sort.Strings(keys)

	c.Startf(fmt.Sprintf("Unsetting %s", strings.Join(keys, ", ")))

	var r *structs.Release

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		for _, e := range c.Args {
			r, err = rack.EnvironmentUnset(app(c), e)
			if err != nil {
				return err
			}
		}
	} else {
		r, err = rack.ReleaseCreate(app(c), structs.ReleaseCreateOptions{Env: options.String(env.String())})
		if err != nil {
			return err
		}
	}

	c.OK()

	c.Writef("Release: <release>%s</release>\n", r.Id)

	if c.Bool("promote") {
		if err := releasePromote(rack, c, app(c), r.Id); err != nil {
			return err
		}
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, r.Id)
	}

	return nil
}
