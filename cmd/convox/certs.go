package main

import (
	"fmt"
	"io/ioutil"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "certs",
		Action:      cmdCertsList,
		Description: "list certificates",
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "upload a certificate",
				Usage:       "<cert.pub> <cert.key>",
				Action:      cmdCertsCreate,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "chain",
						Usage: "intermediate certificate chain",
					},
				},
			},
			{
				Name:        "delete",
				Description: "delete a certificate",
				Usage:       "<id>",
				Action:      cmdCertsDelete,
			},
			{
				Name:        "generate",
				Description: "generate a certificate",
				Usage:       "<domain> [domain...]",
				Action:      cmdCertsGenerate,
			},
		},
	})
}

func cmdCertsList(c *cli.Context) {
	if len(c.Args()) > 0 {
		stdcli.Error(fmt.Errorf("`convox certs` does not take arguments. Perhaps you meant `convox certs generate`?"))
		return
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return
	}

	certs, err := rackClient(c).ListCertificates()
	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("ID", "DOMAIN", "EXPIRES")

	for _, cert := range certs {
		t.AddRow(cert.Id, cert.Domain, humanizeTime(cert.Expiration))
	}

	t.Print()
}

func cmdCertsCreate(c *cli.Context) {
	if len(c.Args()) < 2 {
		stdcli.Usage(c, "create")
		return
	}

	pub, err := ioutil.ReadFile(c.Args()[0])

	if err != nil {
		stdcli.Error(err)
		return
	}

	key, err := ioutil.ReadFile(c.Args()[1])

	if err != nil {
		stdcli.Error(err)
		return
	}

	chain := ""

	if chainFile := c.String("chain"); chainFile != "" {
		data, err := ioutil.ReadFile(chainFile)

		if err != nil {
			stdcli.Error(err)
			return
		}

		chain = string(data)
	}

	fmt.Printf("Uploading certificate... ")

	cert, err := rackClient(c).CreateCertificate(string(pub), string(key), chain)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("OK, %s\n", cert.Id)
}

func cmdCertsDelete(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return
	}

	fmt.Printf("Removing certificate... ")

	err := rackClient(c).DeleteCertificate(c.Args()[0])

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("OK")
}

func cmdCertsGenerate(c *cli.Context) {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "generate")
		return
	}

	fmt.Printf("Requesting certificate... ")

	cert, err := rackClient(c).GenerateCertificate(c.Args())

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("OK, %s\n", cert.Id)
}
