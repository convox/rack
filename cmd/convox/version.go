package main

import (
	"net/url"

	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("version", "display version information", Version, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})
}

func Version(c *stdcli.Context) error {
	c.Writef("client: <info>%s</info>\n", version)

	host, err := currentHost(c)
	if err != nil {
		return err
	}

	ep, err := currentEndpoint(c, currentRack(c, host))
	if err != nil {
		c.Writef("server: <info>none</info>\n")
		return err
	}

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	eu, err := url.Parse(ep)
	if err != nil {
		return err
	}

	c.Writef("server: <info>%s</info> (<info>%s</info>)\n", s.Version, eu.Host)

	return nil
}
