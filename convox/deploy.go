package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "deploy",
		Description: "deploy an app to AWS",
		Usage:       "<directory>",
		Action:      cmdDeploy,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdDeploy(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	_, err = ConvoxGet(fmt.Sprintf("/apps/%s", app))

	if err != nil {
		stdcli.Error(err)
		return
	}

	// build
	release, err := executeBuild(dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	if release == "" {
		return
	}

	a, err := postRelease(app, release)

	if err != nil {
		stdcli.Error(err)
		return
	}

	urls := []string{}
	hosts := []string{}

	matcher := regexp.MustCompile(`^(\w+)Port\d+Balancer`)

	if host, ok := a.Outputs["BalancerHost"]; ok {
		for key, value := range a.Outputs {
			if m := matcher.FindStringSubmatch(key); m != nil {
				url := fmt.Sprintf("http://%s:%s", host, value)
				urls = append(urls, url)
				hosts = append(hosts, fmt.Sprintf("%s: %s", strings.ToLower(m[1]), url))
			}
		}
	}

	fmt.Print("Waiting for app... ")

	ch := make(chan error)

	for _, url := range urls {
		go func() {
			waitForAvailability(url)
			ch <- nil
		}()
	}

	for _ = range urls {
		<-ch
	}

	fmt.Println("OK")

	for _, host := range hosts {
		fmt.Println(host)
	}
}
