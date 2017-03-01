package main

import (
	"fmt"
	"strings"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ssl",
		Action:      cmdSSLList,
		Description: "manage ssl certificates",
		Flags: []cli.Flag{
			appFlag,
			rackFlag,
		},
		Subcommands: []cli.Command{
			{
				Name:        "update",
				Description: "update the certificate associated with an endpoint",
				Usage:       "<process:port> <certificate-id> [options]",
				ArgsUsage:   "<process:port> <certificate-id>",
				Action:      cmdSSLUpdate,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
				},
			},
		},
	})
}

func cmdSSLList(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	ssls, err := rackClient(c).ListSSL(app)
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("TARGET", "CERTIFICATE", "DOMAIN", "EXPIRES")

	for _, ssl := range *ssls {
		t.AddRow(fmt.Sprintf("%s:%d", ssl.Process, ssl.Port), ssl.Certificate, ssl.Domain, helpers.HumanizeTime(ssl.Expiration))
	}

	t.Print()
	return nil
}

func cmdSSLUpdate(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, -2)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		return stdcli.Error(fmt.Errorf("endpoint must be process:port"))
	}

	fmt.Printf("Updating certificate... ")

	_, err = rackClient(c).UpdateSSL(app, parts[0], parts[1], c.Args()[1])
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("OK")
	return nil
}
