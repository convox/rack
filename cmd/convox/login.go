package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/helpers"
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

	stat, err := os.Stat(ConfigRoot)
	if err != nil && !os.IsNotExist(err) {
		stdcli.Error(err)
		return
	}

	if stat != nil && !stat.IsDir() {
		err := upgradeConfig()
		if err != nil {
			stdcli.Error(err)
			return
		}
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

	os.Remove(filepath.Join(ConfigRoot, "rack"))
	os.Remove(filepath.Join(ConfigRoot, "switch"))

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

func upgradeConfig() error {
	data, err := ioutil.ReadFile(ConfigRoot)

	if err != nil {
		return err
	}

	parts := strings.Split(string(data), "\n")

	if len(parts) < 2 {
		return fmt.Errorf("invalid .convox file")
	}

	err = os.Remove(ConfigRoot)

	if err != nil {
		return err
	}

	err = os.MkdirAll(ConfigRoot, 0700)

	if err != nil {
		return err
	}

	err = addLogin(parts[0], parts[1])

	if err != nil {
		return err
	}

	return switchHost(parts[0])
}

func getLogin(host string) (string, error) {
	config := filepath.Join(ConfigRoot, "auth")
	data, _ := ioutil.ReadFile(filepath.Join(config))
	if data == nil {
		data = []byte("{}")
	}

	var auth ConfigAuth
	err := json.Unmarshal(data, &auth)

	if err != nil {
		return "", err
	}

	return auth[host], nil
}

func addLogin(host, password string) error {
	config := filepath.Join(ConfigRoot, "auth")

	data, _ := ioutil.ReadFile(filepath.Join(config))

	if data == nil {
		data = []byte("{}")
	}

	var auth ConfigAuth

	err := json.Unmarshal(data, &auth)

	if err != nil {
		return err
	}

	auth[host] = password

	data, err = json.MarshalIndent(auth, "", "  ")

	if err != nil {
		return err
	}

	err = os.MkdirAll(ConfigRoot, 0755)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(config, data, 0600)
}

func removeLogin(host string) error {
	config := filepath.Join(ConfigRoot, "auth")

	data, _ := ioutil.ReadFile(filepath.Join(config))

	if data == nil {
		data = []byte("{}")
	}

	var auth ConfigAuth

	err := json.Unmarshal(data, &auth)

	if err != nil {
		return err
	}

	delete(auth, host)

	data, err = json.Marshal(auth)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(config, data, 0600)
}

func switchHost(host string) error {
	return ioutil.WriteFile(filepath.Join(ConfigRoot, "host"), []byte(host), 0600)
}

func removeHost() error {
	return os.Remove(filepath.Join(ConfigRoot, "host"))
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

	config := filepath.Join(ConfigRoot, "host")

	if !helpers.Exists(config) {
		return "", fmt.Errorf("no host config found")
	}

	data, err := ioutil.ReadFile(config)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func currentPassword() (string, error) {
	if password := os.Getenv("CONVOX_PASSWORD"); password != "" {
		return password, nil
	}

	config := filepath.Join(ConfigRoot, "auth")

	if !helpers.Exists(config) {
		return "", fmt.Errorf("no auth config found")
	}

	data, err := ioutil.ReadFile(config)

	if err != nil {
		return "", err
	}

	host, err := currentHost()

	if err != nil {
		return "", err
	}

	var auth ConfigAuth

	err = json.Unmarshal(data, &auth)
	if err != nil {
		return "", err
	}

	return auth[host], nil
}

func currentId() (string, error) {
	config := filepath.Join(ConfigRoot, "id")

	if !helpers.Exists(config) {
		err := os.MkdirAll(ConfigRoot, 0700)
		if err != nil {
			return "", err
		}

		id := randomString(20)

		err = ioutil.WriteFile(config, []byte(id), 0600)
		if err != nil {
			return "", err
		}

		return id, nil
	}

	data, err := ioutil.ReadFile(config)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func updateID(id string) error {
	config := filepath.Join(ConfigRoot, "id")

	return ioutil.WriteFile(config, []byte(id), 0600)
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
