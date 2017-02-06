package main

import (
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
)

var Version = "dev"

func init() {
	stdcli.VersionPrinter(func(c *cli.Context) {
		fmt.Printf("client: %s\n", c.App.Version)

		rc := rackClient(c)
		if rc == nil {
			return
		}

		system, err := rc.GetSystem()
		if err != nil {
			stdcli.Error(err)
			return
		}

		host, _, err := currentLogin()
		if err != nil {
			return
		}

		fmt.Printf("server: %s (%s)\n", system.Version, host)
	})
}

func main() {
	app := stdcli.New()
	app.Version = Version
	app.Usage = "command-line application management"

	err := app.Run(os.Args)

	if err != nil {
		if err.Error() == "Token expired" {
			email, err := currentId()
			if err != nil {
				email = promptForUsername()
			} else {
				_, err := mail.ParseAddress(email)
				if err != nil {
					email = promptForUsername()
				}
			}

			pw := promptForPassword()
			host, _ := currentHost()
			cl := client.New(host, "", "")

			token, err := cl.RegenerateToken(email, pw)

			if err == nil {
				err = addLogin(host, token)
				if err != nil {
					stdcli.Error(err)
				}
				err = app.Run(os.Args)
				if err != nil {
					stdcli.Error(err)
					os.Exit(1)
				}
			} else {
				stdcli.Error(err)
				os.Exit(1)
			}
		}
		os.Exit(1)
	}
}

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

// urfave/cli has a habit of discarding flags when they're between a command and a subcommand.
// if --rack is missing from c.String(), recover it here by checking os.Args
func getRackFlag(c *cli.Context) string {
	rackFlag := c.String("rack")
	if rackFlag != "" {
		return rackFlag
	}

	// set rackFlag to everything after --rack
	rackFlag = stdcli.ParseOpts(os.Args)["rack"]

	// stdcli.ParseOpts() includes everything after --rack, so discard everything after the first space
	return strings.Split(rackFlag, " ")[0]
}

func currentRack(c *cli.Context) string {
	cr, err := ioutil.ReadFile(filepath.Join(ConfigRoot, "rack"))
	if err != nil && !os.IsNotExist(err) {
		stdcli.Error(err)
	}

	rackFlag := getRackFlag(c)

	return coalesce(rackFlag, os.Getenv("CONVOX_RACK"), stdcli.ReadSetting("rack"), strings.TrimSpace(string(cr)))
}

func rackClient(c *cli.Context) *client.Client {
	host, password, err := currentLogin()
	if err != nil {
		stdcli.Error(err)
		return nil
	}

	cl := client.New(host, password, c.App.Version)

	cl.Rack = currentRack(c)

	return cl
}
