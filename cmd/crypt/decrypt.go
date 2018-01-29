package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/rack/crypt"

	"gopkg.in/urfave/cli.v1"
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

	dec, err := crypt.New().Decrypt(key, data)

	if err != nil {
		panic(err)
	}

	fmt.Print(string(dec))
}
