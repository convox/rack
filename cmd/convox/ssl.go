package main

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ssl",
		Description: "manage SSL certificates",
		Subcommands: []cli.Command{
			{
				Name:        "add",
				Description: "add a new certificate",
				Usage:       "<foo.crt> <foo.key> [--port=4443]",
				Action:      cmdSSLAdd,
				Flags: []cli.Flag{
					appFlag,
					cli.IntFlag{
						Name:  "port",
						Usage: "non-standard port number, e.g. 4443",
					},
				},
			},
		},
	})
}

func cmdSSLAdd(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 2 {
		stdcli.Usage(c, "add")
		return
	}

	body, err := ioutil.ReadFile(c.Args()[0])

	if err != nil {
		stdcli.Error(err)
		return
	}

	key, err := ioutil.ReadFile(c.Args()[1])

	if err != nil {
		stdcli.Error(err)
		return
	}

	port := 443
	if c.IsSet("port") {
		port = c.Int("port")
	}

	p := strconv.Itoa(port)

	fmt.Printf("Adding SSL to %s (%s)... ", app, port)

	_, err = rackClient(c).CreateSSL(app, string(body), string(key), p)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}
