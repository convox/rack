package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
	homedir "github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v1"
)

var ConfigRoot string

type ConfigAuth map[string]string

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "login",
		Description: "log into your convox rack",
		Usage:       "[hostname]",
		Action:      cmdLogin,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "password, p",
				Usage: "Console API key or Rack password. If not specified, prompt.",
			},
		},
	})

	home, err := homedir.Dir()
	if err != nil {
		return
	}

	ConfigRoot = filepath.Join(home, ".convox")

	if root := os.Getenv("CONVOX_CONFIG"); root != "" {
		ConfigRoot = root
	}
}

func cmdLogin(c *cli.Context) error {
	var host string

	if len(c.Args()) < 1 {
		host = "console.convox.com"
	} else {
		host = c.Args()[0]
	}

	u, err := url.Parse(host)
	if err != nil {
		return stdcli.Error(err)
	}

	if u.Host != "" {
		host = u.Host
	}

	password := os.Getenv("CONVOX_PASSWORD")
	if password == "" {
		password = c.String("password")
	}

	var auth *client.Auth

	if password != "" {
		// password flag
		auth, err = testLogin(host, password, c.App.Version)
	} else {
		// first try current login
		password, err = getLogin(host)
		auth, err = testLogin(host, password, c.App.Version)
		// then prompt for password
		if err != nil {
			password = promptForPassword()
			auth, err = testLogin(host, password, c.App.Version)
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "401") {
			return stdcli.Error(fmt.Errorf("invalid login\nHave you created an account at https://convox.com/signup?"))
		} else {
			return stdcli.Error(err)
		}
	}

	err = addLogin(host, password)
	if err != nil {
		return stdcli.Error(err)
	}

	if auth.ID != "" {
		updateID(auth.ID)
	}

	removeConfig("rack")
	removeConfig("switch")

	err = switchHost(host)
	if err != nil {
		return stdcli.Error(err)
	}

	distinctID, err = currentId()
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("Logged in successfully.")
	return nil
}

func readAuth() (ConfigAuth, error) {
	data := readConfig("auth")
	if data == "" {
		return ConfigAuth{}, nil
	}

	var auth ConfigAuth

	if err := json.Unmarshal([]byte(data), &auth); err != nil {
		return nil, err
	}

	return auth, nil
}

func writeAuth(auth ConfigAuth) error {
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	return writeConfig("auth", string(data))
}

func getLogin(host string) (string, error) {
	auth, err := readAuth()
	if err != nil {
		return "", err
	}

	return auth[host], nil
}

func addLogin(host, password string) error {
	auth, err := readAuth()
	if err != nil {
		return err
	}

	auth[host] = password

	if err := writeAuth(auth); err != nil {
		return err
	}

	return nil
}

func removeLogin(host string) error {
	auth, err := readAuth()
	if err != nil {
		return err
	}

	delete(auth, host)

	if err := writeAuth(auth); err != nil {
		return err
	}

	return nil
}

func switchHost(host string) error {
	return writeConfig("host", host)
}

func removeHost() error {
	return removeConfig("host")
}

func currentLogin() (string, string, error) {
	host, err := currentHost()
	if err != nil {
		return "", "", err
	}

	password, err := currentPassword()
	if err != nil {
		return "", "", err
	}

	return host, password, nil
}

func currentHost() (string, error) {
	if host := os.Getenv("CONVOX_HOST"); host != "" {
		return host, nil
	}

	host := strings.TrimSpace(readConfig("host"))

	if host == "" {
		return "", fmt.Errorf("no host config found")
	}

	return host, nil
}

func currentPassword() (string, error) {
	if password := os.Getenv("CONVOX_PASSWORD"); password != "" {
		return password, nil
	}

	host, err := currentHost()
	if err != nil {
		return "", err
	}

	password, err := getLogin(host)
	if err != nil {
		return "", err
	}

	return password, nil
}

func currentId() (string, error) {
	id := readConfig("id")

	if id == "" {
		id = randomString(20)

		if err := writeConfig("id", id); err != nil {
			return "", err
		}
	}

	return strings.TrimSpace(id), nil
}

func updateID(id string) error {
	return writeConfig("id", strings.TrimSpace(id))
}

func testLogin(host, password, version string) (*client.Auth, error) {
	return client.New(host, password, version).Auth()
}

func promptForPassword() string {
	fmt.Print("Password: ")

	in, err := terminal.ReadPassword(int(os.Stdin.Fd()))

	fmt.Println()

	if err != nil {
		stdcli.Error(err)
		return ""
	}

	return string(in)
}

func promptForUsername() string {
	fmt.Print("Username: ")

	var in string
	_, err := fmt.Scanln(&in)

	if err != nil {
		stdcli.Error(err)
		return ""
	}

	return in
}
