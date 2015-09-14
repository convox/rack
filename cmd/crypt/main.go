package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/api/crypt"
)

func main() {
	app := NewCli()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "role",
			Usage:  "iam role",
			EnvVar: "AWS_ROLE",
		},
		cli.StringFlag{
			Name:   "region",
			Usage:  "aws region",
			EnvVar: "AWS_REGION",
		},
		cli.StringFlag{
			Name:   "access",
			Usage:  "aws access id",
			EnvVar: "AWS_ACCESS",
		},
		cli.StringFlag{
			Name:   "secret",
			Usage:  "aws secret key",
			EnvVar: "AWS_SECRET",
		},
	}

	app.Usage = "env management"

	app.Run(os.Args)
}

func buildCrypt(c *cli.Context) (*crypt.Crypt, error) {
	if role := c.GlobalString("role"); role != "" {
		return crypt.NewIam(role)
	} else {
		region := c.GlobalString("region")
		access := c.GlobalString("access")
		secret := c.GlobalString("secret")

		return crypt.New(region, access, secret), nil
	}
}
