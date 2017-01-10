package main

import (
	"fmt"
	"strings"

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
				Usage:       "<process:port> <certificate-id>",
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
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) > 0 {
		return stdcli.Error(fmt.Errorf("`convox ssl` does not take arguments. Perhaps you meant `convox ssl update`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	ssls, err := rackClient(c).ListSSL(app)
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("TARGET", "CERTIFICATE", "DOMAIN", "EXPIRES")

	for _, ssl := range *ssls {
		t.AddRow(fmt.Sprintf("%s:%d", ssl.Process, ssl.Port), ssl.Certificate, ssl.Domain, humanizeTime(ssl.Expiration))
	}

	t.Print()
	return nil
}

func cmdSSLUpdate(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "update")
		return nil
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
