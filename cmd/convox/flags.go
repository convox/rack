package main

import "github.com/codegangsta/cli"

var appFlag = cli.StringFlag{
	Name:  "app, a",
	Usage: "App name. Inferred from current directory if not specified.",
}
