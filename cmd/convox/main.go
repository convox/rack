package main

import (
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/sdk"
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

	terminalSetup()

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

type Rack struct {
	Host   string
	Name   string
	Status string
}

type Racks []Rack

func currentRack(c *cli.Context) string {
	cr, err := ioutil.ReadFile(filepath.Join(ConfigRoot, "rack"))
	if err != nil && !os.IsNotExist(err) {
		stdcli.Error(err)
	}

	rackFlag := stdcli.RecoverFlag(c, "rack")

	return helpers.Coalesce(rackFlag, os.Getenv("CONVOX_RACK"), stdcli.ReadSetting("rack"), strings.TrimSpace(string(cr)))
}

func rack(c *cli.Context) *sdk.Client {
	cr := currentRack(c)

	if cr == "local" {
		if !localRackRunning() {
			stdcli.Errorf("local rack is not running")
			os.Exit(1)
		}

		cl, err := sdk.New("https://rack.convox")
		if err != nil {
			stdcli.Error(err)
			os.Exit(1)
		}

		return cl
	}

	host, password, err := currentLogin()
	if err != nil {
		stdcli.Errorf("%s, try `convox login`", err)
		os.Exit(1)
	}

	cl, err := sdk.New(fmt.Sprintf("https://%s@%s", password, host))
	if err != nil {
		stdcli.Error(err)
		os.Exit(1)
	}

	cl.Rack = cr

	return cl
}

func rackClient(c *cli.Context) *client.Client {
	if os.Getenv("CONVOX_HOST") != "" {
		return client.New(os.Getenv("CONVOX_HOST"), os.Getenv("CONVOX_PASSWORD"), c.App.Version)
	}

	name := currentRack(c)

	if c.String("rack") != "" {
		name = c.String("rack")
	}

	if name == "" {
		stdcli.Errorf("no host config found, try `convox login`")
		os.Exit(1)
	}

	rack, err := rackGet(currentRack(c))
	if err != nil {
		stdcli.Errorf("could not get rack: %s", currentRack(c))
		os.Exit(1)
	}

	password, _ := getLogin(rack.Host)

	cl := client.New(rack.Host, password, c.App.Version)
	cl.Rack = rack.Name

	return cl
}

func rackGet(name string) (*Rack, error) {
	racks := rackList()

	for _, r := range racks {
		if r.Name == name {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("no such rack: %s", name)
}

func rackList() Racks {
	racks := Racks{}

	rrs, err := remoteRacks()
	if err == nil {
		racks = append(racks, rrs...)
	}

	lrs, err := localRacks()
	if err == nil {
		racks = append(racks, lrs...)
	}

	sort.Slice(racks, func(i, j int) bool {
		return racks[i].Name < racks[j].Name
	})

	return racks
}

func remoteRacks() (Racks, error) {
	host, password, err := currentLogin()
	if err != nil {
		return nil, err
	}

	c := client.New(host, password, Version)

	rs, err := c.Racks()
	if err != nil {
		return nil, err
	}

	racks := make(Racks, len(rs))

	for i, r := range rs {
		name := r.Name

		if r.Organization.Name != "" {
			name = fmt.Sprintf("%s/%s", r.Organization.Name, r.Name)
		}

		racks[i] = Rack{
			Host:   host,
			Name:   name,
			Status: r.Status,
		}
	}

	return racks, nil
}

func localRacks() (Racks, error) {
	data, err := exec.Command("docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").CombinedOutput()
	if err != nil {
		return nil, err
	}

	names := strings.Split(strings.TrimSpace(string(data)), "\n")

	racks := make(Racks, len(names))

	for i, name := range names {
		racks[i] = Rack{
			Host:   fmt.Sprintf("rack.%s", name),
			Name:   fmt.Sprintf("local/%s", name),
			Status: "running",
		}
	}

	return racks, nil
}
