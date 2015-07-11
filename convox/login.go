package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	homedir "github.com/convox/cli/Godeps/_workspace/src/github.com/mitchellh/go-homedir"
	"github.com/convox/cli/Godeps/_workspace/src/golang.org/x/crypto/ssh/terminal"
	"github.com/convox/cli/stdcli"
)

var ConfigRoot string

type ConfigAuth map[string]string

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "login",
		Description: "login to your convox installation",
		Usage:       "<host>",
		Action:      cmdLogin,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "password",
				Usage: "password to use for authentication. If not specified, prompt for password.",
			},
		},
	})

	home, err := homedir.Dir()

	if err != nil {
		log.Fatal(err)
	}

	ConfigRoot = filepath.Join(home, ".convox")

	stat, err := os.Stat(ConfigRoot)

	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	if stat != nil && !stat.IsDir() {
		err := upgradeConfig()

		if err != nil {
			log.Fatal(err)
		}
	}
}

func cmdLogin(c *cli.Context) {
	if len(c.Args()) != 1 {
		stdcli.Usage(c, "login")
	}

	host := c.Args()[0]
	u, err := url.Parse(host)

	if err != nil {
		stdcli.Error(err)
		return
	}

	if u.Host != "" {
		host = u.Host
	}

	password := c.String("password")

	if password == "" {
		fmt.Print("Password: ")

		in, err := terminal.ReadPassword(int(os.Stdin.Fd()))

		fmt.Println()

		if err != nil {
			stdcli.Error(err)
			return
		}

		password = string(in)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/apps", host), nil)

	if err != nil {
		stdcli.Error(err)
		return
	}

	req.SetBasicAuth("convox", string(password))

	res, err := client.Do(req)

	if err != nil {
		stdcli.Error(err)
		return
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		stdcli.Error(fmt.Errorf("invalid login"))
		return
	}

	err = addLogin(host, password)

	if err != nil {
		stdcli.Error(err)
		return
	}

	err = switchHost(host)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println("Logged in successfully.")
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

	err = switchHost(parts[0])

	if err != nil {
		return err
	}

	return nil
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

	data, err = json.Marshal(auth)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(config, data, 0600)
}

func switchHost(host string) error {
	return ioutil.WriteFile(filepath.Join(ConfigRoot, "host"), []byte(host), 0600)
}

func currentLogin() (string, string, error) {
	host, err := currentHost()

	if err != nil {
		return "", "", fmt.Errorf("must login first")
	}

	password, err := currentPassword()

	if err != nil {
		return "", "", fmt.Errorf("must login first")
	}

	return host, password, nil
}

func currentHost() (string, error) {
	if host := os.Getenv("CONVOX_HOST"); host != "" {
		return host, nil
	}

	config := filepath.Join(ConfigRoot, "host")

	if !exists(config) {
		return "", fmt.Errorf("no host config")
	}

	data, err := ioutil.ReadFile(config)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func currentPassword() (string, error) {
	if password := os.Getenv("CONVOX_PASSWORD"); password != "" {
		return password, nil
	}

	config := filepath.Join(ConfigRoot, "auth")

	if !exists(config) {
		return "", fmt.Errorf("no auth config")
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

	return auth[host], nil
}
