package main

import (
	"fmt"
	"io/ioutil"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "ssl",
		Action:      cmdSSLList,
		Description: "manage SSL certificates",
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new SSL listener",
				Usage:       "<port> <foo.crt> <foo.key>",
				Action:      cmdSSLCreate,
				Flags: []cli.Flag{
					appFlag,
				},
			},
			{
				Name:        "delete",
				Description: "delete an SSL listener",
				Usage:       "<port>",
				Action:      cmdSSLDelete,
				Flags: []cli.Flag{
					appFlag,
				},
			},
		},
	})
}

func cmdSSLCreate(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 3 {
		stdcli.Usage(c, "add")
		return
	}

	port := c.Args()[0]

	body, err := ioutil.ReadFile(c.Args()[1])

	if err != nil {
		stdcli.Error(err)
		return
	}

	key, err := ioutil.ReadFile(c.Args()[2])

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Creating SSL listener on port %s... ", port)

	_, err = rackClient(c).CreateSSL(app, port, string(body), string(key))

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
}

func cmdSSLDelete(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return
	}

	port := c.Args()[0]

	fmt.Printf("Deleting SSL listener on port %s... ", port)

	_, err = rackClient(c).DeleteSSL(app, port)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Done.")
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

	t := stdcli.NewTable("PORT", "EXPIRES", "DOMAINS")

	for _, ssl := range *ssls {
		t.AddRow(ssl.Port, humanizeTime(ssl.Expiration), ssl.Name)
	}

	t.Print()
}
