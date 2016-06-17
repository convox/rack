package main

import "gopkg.in/urfave/cli.v1"

var appFlag = cli.StringFlag{
	Name:  "app, a",
	Usage: "App name. Inferred from current directory if not specified.",
}

var rackFlag = cli.StringFlag{
	Name:  "rack",
	Usage: "Rack name.",
}
