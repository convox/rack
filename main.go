package main

import (
	"os"

	"github.com/convox/env/Godeps/_workspace/src/github.com/codegangsta/cli"
)

func main() {
	app := NewCli()

	app.Flags = []cli.Flag{
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
