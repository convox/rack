package main

import (
	"fmt"
	"io/ioutil"

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
				Usage:       "<id>",
				Action:      cmdCertsDelete,
				Flags:       []cli.Flag{rackFlag},
			},
			{
				Name:        "generate",
				Description: "generate a certificate",
				Usage:       "<domain> [domain...]",
				Action:      cmdCertsGenerate,
				Flags:       []cli.Flag{rackFlag},
			},
		},
	})
}

func cmdCertsList(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox certs` does not take arguments. Perhaps you meant `convox certs generate`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	certs, err := rackClient(c).ListCertificates()
	if err != nil {
		return stdcli.ExitError(err)
	}

	t := stdcli.NewTable("ID", "DOMAIN", "EXPIRES")

	for _, cert := range certs {
		t.AddRow(cert.Id, cert.Domain, humanizeTime(cert.Expiration))
	}

	t.Print()
	return nil
}

func cmdCertsCreate(c *cli.Context) error {
	if len(c.Args()) < 2 {
		stdcli.Usage(c, "create")
		return nil
	}

	pub, err := ioutil.ReadFile(c.Args()[0])
	if err != nil {
		return stdcli.ExitError(err)
	}

	key, err := ioutil.ReadFile(c.Args()[1])
	if err != nil {
		return stdcli.ExitError(err)
	}

	chain := ""

	if chainFile := c.String("chain"); chainFile != "" {
		data, err := ioutil.ReadFile(chainFile)
		if err != nil {
			return stdcli.ExitError(err)
		}

		chain = string(data)
	}

	fmt.Printf("Uploading certificate... ")

	cert, err := rackClient(c).CreateCertificate(string(pub), string(key), chain)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("OK, %s\n", cert.Id)
	return nil
}

func cmdCertsDelete(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	fmt.Printf("Removing certificate... ")

	err := rackClient(c).DeleteCertificate(c.Args()[0])
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("OK")
	return nil
}

func cmdCertsGenerate(c *cli.Context) error {
	if len(c.Args()) < 1 {
		stdcli.Usage(c, "generate")
		return nil
	}

	fmt.Printf("Requesting certificate... ")

	cert, err := rackClient(c).GenerateCertificate(c.Args())
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("OK, %s\n", cert.Id)
	return nil
}
