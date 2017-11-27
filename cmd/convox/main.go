package main

import (
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"gopkg.in/urfave/cli.v1"

	"github.com/Azure/go-ansiterm/winterm"
	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/helpers"
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

/*
	## Command syntax ##

		Usage:
			no more than one line

		UsageText:
			may be multiple lines, but isn't usually displayed

		ArgsUsage:
			no more than one line
			denotes required arguments
			used in output of stdcli.NeedArg as placeholder when argument is missing or unexpected

		Description:
			no more than one line
			used in:
				* 'convox <command> --help' under 'Usage:'
				* output of 'convox -h' and 'convox h'
*/

func main() {
	app := stdcli.New()
	app.Flags = []cli.Flag{appFlag, rackFlag}
	app.Version = Version

	if runtime.GOOS == "windows" {
		enableWindowsNativeTerminal()
	}

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

		if _, ok := err.(stdcli.ErrorStdCli); !ok {
			stdcli.Error(err)
		}
		os.Exit(1)
	}
}

func currentRack(c *cli.Context) string {
	cr, err := ioutil.ReadFile(filepath.Join(ConfigRoot, "rack"))
	if err != nil && !os.IsNotExist(err) {
		stdcli.Error(err)
	}

	rackFlag := stdcli.RecoverFlag(c, "rack")

	return helpers.Coalesce(rackFlag, os.Getenv("CONVOX_RACK"), stdcli.ReadSetting("rack"), strings.TrimSpace(string(cr)))
}

func rackClient(c *cli.Context) *client.Client {
	host, password, err := currentLogin()
	if err != nil {
		stdcli.Errorf("%s, try `convox login`", err)
		os.Exit(1)
	}

	cl := client.New(host, password, c.App.Version)
	cl.Rack = currentRack(c)

	return cl
}

const enableVirtualTerminalProcessing = 0x0004

func enableWindowsNativeTerminal() error {
	hnd, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return err
	}

	var mode uint32

	if err := syscall.GetConsoleMode(hnd, &mode); err != nil {
		return err
	}

	if err := winterm.SetConsoleMode(uintptr(hnd), mode|enableVirtualTerminalProcessing); err != nil {
		return err
	}

	return nil
}
