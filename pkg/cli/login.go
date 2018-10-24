package cli

import (
	"fmt"
	"net/url"
	"os"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	registerWithoutProvider("login", "authenticate with a rack", Login, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			stdcli.StringFlag("password", "p", "password"),
		},
		Usage:    "[hostname]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Login(rack sdk.Interface, c *stdcli.Context) error {
	hostname := coalesce(c.Arg(0), "console.convox.com")

	auth, err := hostAuth(c, hostname)
	if err != nil {
		return err
	}

	password := coalesce(c.String("password"), os.Getenv("CONVOX_PASSWORD"), auth)

	if password == "" {
		c.Writef("Password: ")

		password, err = c.ReadSecret()
		if err != nil {
			return err
		}

		c.Writef("\n")
	}

	c.Startf("Authenticating with <info>%s</info>", hostname)

	cl, err := sdk.New(fmt.Sprintf("https://convox:%s@%s", url.QueryEscape(password), hostname))
	if err != nil {
		return err
	}

	id, err := cl.Auth()
	if err != nil {
		return fmt.Errorf("invalid login")
	}

	if err := saveAuth(c, hostname, password); err != nil {
		return err
	}

	if err := c.SettingWrite("host", hostname); err != nil {
		return err
	}

	if id != "" {
		if err := c.SettingWrite("id", id); err != nil {
			return err
		}
	}

	if err := c.SettingDelete("rack"); err != nil {
		return err
	}

	return c.OK()
}
