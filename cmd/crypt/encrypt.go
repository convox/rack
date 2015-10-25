package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
)

func init() {
	RegisterCommand(cli.Command{
		Name:        "encrypt",
		Description: "encrypt an env",
		Usage:       "<key> [filename]",
		Action:      cmdEncrypt,
	})
}

func cmdEncrypt(c *cli.Context) {
	if len(c.Args()) < 1 {
		Usage(c, "encrypt")
		return
	}

	key := c.Args()[0]

	var env []byte
	var err error

	if len(c.Args()) == 1 {
		env, err = ioutil.ReadAll(os.Stdin)
	} else {
		env, err = ioutil.ReadFile(c.Args()[1])
	}

	if err != nil {
		panic(err)
	}

	cr, err := buildCrypt(c)

	if err != nil {
		panic(err)
	}

	data, err := cr.Encrypt(key, env)

	if err != nil {
		panic(err)
	}

	fmt.Print(string(data))
}
