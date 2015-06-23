package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
	"golang.org/x/crypto/ssh/terminal"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "login",
		Description: "login to your convox installation",
		Usage:       "<host>",
		Action:      cmdLogin,
	})
}

func cmdLogin(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "login")
	}

	host := c.Args()[0]

	fmt.Print("Password: ")

	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))

	fmt.Println()

	if err != nil {
		stdcli.Error(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/apps", host), nil)

	if err != nil {
		stdcli.Error(err)
	}

	req.SetBasicAuth("convox", string(password))

	res, err := client.Do(req)

	if err != nil {
		stdcli.Error(err)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		stdcli.Error(fmt.Errorf("invalid login"))
	}

	u, err := user.Current()

	err = ioutil.WriteFile(filepath.Join(u.HomeDir, ".convox"), []byte(fmt.Sprintf("%s\n%s\n", host, password)), 0600)

	if err != nil {
		stdcli.Error(err)
	}

	fmt.Println("Login Succeeded")
}
