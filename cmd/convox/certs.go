package main

import (
	"fmt"
	"io/ioutil"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "certs",
		Action:      cmdCertsList,
		Description: "list certificates",
		Flags:       []cli.Flag{rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "upload a certificate",
				Usage:       "<cert.pub> <cert.key>",
				ArgsUsage:   "<cert.pub> <cert.key>",
				Action:      cmdCertsCreate,
				Flags: []cli.Flag{
					rackFlag,
					cli.StringFlag{
						Name:  "chain",
						Usage: "intermediate certificate chain",
					},
				},
			},
			{
				Name:        "delete",
				Description: "delete a certificate",
				Usage:       "<cert id>",
				ArgsUsage:   "<cert id>",
				Action:      cmdCertsDelete,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "generate",
				Description: "generate a certificate",
				Usage:       "<domain> [domain...]",
				ArgsUsage:   "<domain> [domain...]",
				Action:      cmdCertsGenerate,
				Flags:       []cli.Flag{rackFlag},
			},
		},
	})
}

func cmdCertsList(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	certs, err := rackClient(c).ListCertificates()
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("ID", "DOMAIN", "EXPIRES")

	for _, cert := range certs {
		t.AddRow(cert.Id, cert.Domain, helpers.HumanizeTime(cert.Expiration))
	}

	t.Print()
	return nil
}

func cmdCertsCreate(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 2)

	pub, err := ioutil.ReadFile(c.Args()[0])
	if err != nil {
		return stdcli.Error(err)
	}

	key, err := ioutil.ReadFile(c.Args()[1])
	if err != nil {
		return stdcli.Error(err)
	}

	chain := ""

	if chainFile := c.String("chain"); chainFile != "" {
		data, err := ioutil.ReadFile(chainFile)
		if err != nil {
			return stdcli.Error(err)
		}

		chain = string(data)
	}

	fmt.Printf("Uploading certificate... ")

	cert, err := rackClient(c).CreateCertificate(string(pub), string(key), chain)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("OK, %s\n", cert.Id)
	return nil
}

func cmdCertsDelete(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	fmt.Printf("Removing certificate... ")

	err := rackClient(c).DeleteCertificate(c.Args()[0])
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("OK")
	return nil
}

func cmdCertsGenerate(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, -1)

	fmt.Printf("Requesting certificate... ")

	cert, err := rackClient(c).GenerateCertificate(c.Args())
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("OK, %s\n", cert.Id)
	return nil
}
