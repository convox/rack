package cli

import (
	"net/url"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("version", "display version information", Version, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})
}

func Version(rack sdk.Interface, c *stdcli.Context) error {
	c.Writef("client: <info>%s</info>\n", c.Version())

	host, err := currentHost(c)
	if err != nil {
		c.Writef("server: <info>none</info>\n")
		return nil
	}

	ep, err := currentEndpoint(c, currentRack(c, host))
	if err != nil {
		c.Writef("server: <info>none</info>\n")
		return nil
	}

	s, err := rack.SystemGet()
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
