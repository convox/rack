package main

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("certs", "list certificates", Certs, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	CLI.Command("certs delete", "delete a certificate", CertsDelete, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<cert>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("certs generate", "generate a certificate", CertsGenerate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagId, flagRack},
		Usage:    "<domain> [domain...]",
		Validate: stdcli.ArgsMin(1),
	})

	CLI.Command("certs import", "import a certificate", CertsImport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagId,
			flagRack,
			stdcli.StringFlag("chain", "", "intermediate certificate chain"),
		},
		Usage:    "<pub> <key>",
		Validate: stdcli.Args(2),
	})
}

func Certs(c *stdcli.Context) error {
	cs, err := provider(c).CertificateList()
	if err != nil {
		return err
	}

	t := c.Table("ID", "DOMAIN", "EXPIRES")

	for _, c := range cs {
		t.AddRow(c.Id, c.Domain, helpers.Ago(c.Expiration))
	}

	return t.Print()
}

func CertsDelete(c *stdcli.Context) error {
	cert := c.Arg(0)

	c.Startf("Deleting certificate <id>%s</id>", cert)

	if err := provider(c).CertificateDelete(cert); err != nil {
		return err
	}

	return c.OK()
}

func CertsGenerate(c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	c.Startf("Generating certificate")

	cr, err := provider(c).CertificateGenerate(c.Args)
	if err != nil {
		return err
	}

	c.OK(cr.Id)

	if c.Bool("id") {
		fmt.Fprintf(stdout, cr.Id)
	}

	return nil
}

func CertsImport(c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	pub, err := ioutil.ReadFile(c.Arg(0))
	if err != nil {
		return err
	}

	key, err := ioutil.ReadFile(c.Arg(1))
	if err != nil {
		return err
	}

	var opts structs.CertificateCreateOptions

	if cf := c.String("chain"); cf != "" {
		chain, err := ioutil.ReadFile(cf)
		if err != nil {
			return err
		}

		opts.Chain = options.String(string(chain))
	}

	c.Startf("Importing certificate")

	var cr *structs.Certificate

	if s.Version <= "20180708231844" {
		cr, err = provider(c).CertificateCreateClassic(string(pub), string(key), opts)
		if err != nil {
			return err
		}
	} else {
		cr, err = provider(c).CertificateCreate(string(pub), string(key), opts)
		if err != nil {
			return err
		}
	}

	c.OK(cr.Id)

	if c.Bool("id") {
		fmt.Fprintf(stdout, cr.Id)
	}

	return nil
}
