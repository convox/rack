package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("env", "list env vars", Env, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Validate: stdcli.Args(0),
	})

	CLI.Command("env get", "get an env var", EnvGet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<var>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("env set", "set env var(s)", EnvSet, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagId,
			flagRack,
			flagWait,
			stdcli.BoolFlag("promote", "p", "promote the release"),
		},
		Usage: "<key=value> [key=value]...",
	})

	CLI.Command("env unset", "unset env var(s)", EnvUnset, stdcli.CommandOptions{
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

func Env(c *stdcli.Context) error {
	env, err := helpers.AppEnvironment(provider(c), app(c))
	if err != nil {
		return err
	}

	c.Writef("%s\n", env.String())

	return nil
}

func EnvGet(c *stdcli.Context) error {
	env, err := helpers.AppEnvironment(provider(c), app(c))
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

func EnvSet(c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	env, err := helpers.AppEnvironment(provider(c), app(c))
	if err != nil {
		return err
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

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		r, err = provider(c).EnvironmentSet(app(c), []byte(env.String()))
		if err != nil {
			return err
		}
	} else {
		r, err = provider(c).ReleaseCreate(app(c), structs.ReleaseCreateOptions{Env: options.String(env.String())})
		if err != nil {
			return err
		}
	}

	c.OK()

	c.Writef("Release: <release>%s</release>\n", r.Id)

	if c.Bool("promote") {
		if err := releasePromote(c, app(c), r.Id); err != nil {
			return err
		}
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, r.Id)
	}

	return nil
}

func EnvUnset(c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	env, err := helpers.AppEnvironment(provider(c), app(c))
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

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	if s.Version <= "20180708231844" {
		for _, e := range c.Args {
			r, err = provider(c).EnvironmentUnset(app(c), e)
			if err != nil {
				return err
			}
		}
	} else {
		r, err = provider(c).ReleaseCreate(app(c), structs.ReleaseCreateOptions{Env: options.String(env.String())})
		if err != nil {
			return err
		}
	}

	c.OK()

	c.Writef("Release: <release>%s</release>\n", r.Id)

	if c.Bool("promote") {
		if err := releasePromote(c, app(c), r.Id); err != nil {
			return err
		}
	}

	if c.Bool("id") {
		fmt.Fprintf(stdout, r.Id)
	}

	return nil
}
