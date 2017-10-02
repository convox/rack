package main

import "gopkg.in/urfave/cli.v1"

var appFlag = cli.StringFlag{
	Name:  "app, a",
	Usage: "app name inferred from current directory if not specified",
}

var rackFlag = cli.StringFlag{
	Name:  "rack",
	Usage: "rack name",
}

var waitFlag = cli.BoolFlag{
	Name:   "wait",
	EnvVar: "CONVOX_WAIT",
	Usage:  "wait for change to finish before returning",
}
