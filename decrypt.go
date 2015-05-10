package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/env/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/env/crypt"
)

func init() {
	RegisterCommand(cli.Command{
		Name:        "decrypt",
		Description: "decrypt an env",
		Usage:       "<key> [filename]",
		Action:      cmdDecrypt,
	})
}

func cmdDecrypt(c *cli.Context) {
	if len(c.Args()) < 1 {
		Usage(c, "decrypt")
		return
	}

	key := c.Args()[0]

	var data []byte
	var err error

	if len(c.Args()) == 1 {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(c.Args()[1])
	}

	if err != nil {
		panic(err)
	}

	e, err := crypt.UnmarshalEnvelope(data)

	if err != nil {
		panic(err)
	}

	cr := &crypt.Crypt{
		AwsRegion: c.GlobalString("region"),
		AwsAccess: c.GlobalString("access"),
		AwsSecret: c.GlobalString("secret"),
	}

	dec, err := cr.Decrypt(key, e)

	if err != nil {
		panic(err)
	}

	fmt.Print(string(dec))
}
