package main

import (
	"fmt"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ssl",
		Action:      cmdSSLList,
		Description: "manage SSL certificates",
		Flags: []cli.Flag{
			appFlag,
		},
		Subcommands: []cli.Command{
			{
				Name:        "update",
				Description: "upload a replacement ssl certificate",
				Usage:       "<process:port> <certificate>",
				Action:      cmdSSLUpdate,
				Flags: []cli.Flag{
					appFlag,
					cli.StringFlag{
						Name:  "chain",
						Usage: "Intermediate certificate chain.",
					},
				},
			},
		},
	})
}

func cmdSSLList(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	ssls, err := rackClient(c).ListSSL(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("TARGET", "CERTIFICATE", "DOMAIN", "EXPIRES")

	for _, ssl := range *ssls {
		t.AddRow(fmt.Sprintf("%s:%d", ssl.Process, ssl.Port), ssl.Certificate, ssl.Domain, humanizeTime(ssl.Expiration))
	}

	t.Print()
}

func cmdSSLUpdate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "update")
		return
	}

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		stdcli.Error(fmt.Errorf("target must be process:port"))
		return
	}

	fmt.Printf("Updating SSL certificate... ")

	_, err = rackClient(c).UpdateSSL(app, parts[0], parts[1], c.Args()[1])

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}
