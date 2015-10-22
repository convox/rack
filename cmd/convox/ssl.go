package main

import (
	"fmt"
	"io/ioutil"
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
				Name:        "create",
				Description: "create a new SSL listener",
				Usage:       "<process:port> <foo.crt> <foo.key>",
				Action:      cmdSSLCreate,
				Flags: []cli.Flag{
					appFlag,
				},
			},
			{
				Name:        "delete",
				Description: "delete an SSL listener",
				Usage:       "<process:port>",
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
		stdcli.Usage(c, "create")
		return
	}

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		stdcli.Error(fmt.Errorf("target must be process:port"))
		return
	}

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

	fmt.Printf("Creating SSL listener %s... ", target)

	_, err = rackClient(c).CreateSSL(app, parts[0], parts[1], string(body), string(key))

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

	target := c.Args()[0]

	parts := strings.Split(target, ":")

	if len(parts) != 2 {
		stdcli.Error(fmt.Errorf("target must be process:port"))
		return
	}

	fmt.Printf("Deleting SSL listener %s... ", target)

	_, err = rackClient(c).DeleteSSL(app, parts[0], parts[1])

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

	t := stdcli.NewTable("TARGET", "EXPIRES", "DOMAINS")

	for _, ssl := range *ssls {
		t.AddRow(fmt.Sprintf("%s:%d", ssl.Process, ssl.Port), humanizeTime(ssl.Expiration), ssl.Name)
	}

	t.Print()
}
