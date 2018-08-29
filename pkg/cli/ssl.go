package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("ssl", "list certificate associates for an app", Ssl, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Validate: stdcli.Args(0),
	})

	register("ssl update", "update certificate for an app", SslUpdate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp, flagWait},
		Usage:    "<process:port> <certificate>",
		Validate: stdcli.Args(2),
	})
}

func Ssl(rack sdk.Interface, c *stdcli.Context) error {
	sys, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var ss structs.Services

	if sys.Version < "20180708231844" {
		ss, err = rack.FormationGet(app(c))
		if err != nil {
			return err
		}
	} else {
		ss, err = rack.ServiceList(app(c))
		if err != nil {
			return err
		}
	}

	t := c.Table("ENDPOINT", "CERTIFICATE", "DOMAIN", "EXPIRES")

	certs := map[string]structs.Certificate{}

	cs, err := rack.CertificateList()
	if err != nil {
		return err
	}

	for _, c := range cs {
		certs[c.Id] = c
	}

	for _, s := range ss {
		for _, p := range s.Ports {
			if p.Certificate != "" {
				t.AddRow(fmt.Sprintf("%s:%d", s.Name, p.Balancer), p.Certificate, certs[p.Certificate].Domain, helpers.Ago(certs[p.Certificate].Expiration))
			}
		}
	}

	return t.Print()
}

func SslUpdate(rack sdk.Interface, c *stdcli.Context) error {
	a, err := rack.AppGet(app(c))
	if err != nil {
		return err
	}

	if a.Generation == "2" {
		return fmt.Errorf("command not valid for generation 2 applications")
	}

	parts := strings.SplitN(c.Arg(0), ":", 2)

	if len(parts) != 2 {
		return fmt.Errorf("process:port required as first argument")
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	c.Startf("Updating certificate")

	if err := rack.CertificateApply(app(c), parts[0], port, c.Arg(1)); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppRunning(rack, c, app(c)); err != nil {
			return err
		}
	}

	return c.OK()
}
