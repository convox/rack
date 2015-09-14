package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/convox/rack/cmd/convox/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/Godeps/_workspace/src/github.com/dustin/go-humanize"
	"github.com/convox/rack/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "builds",
		Description: "manage an app's builds",
		Usage:       "",
		Action:      cmdBuilds,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new build",
				Usage:       "",
				Action:      cmdBuildsCreate,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "info",
				Description: "print output for a build",
				Usage:       "<ID>",
				Action:      cmdBuildsInfo,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdBuilds(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	builds, err := rackClient(c).GetBuilds(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	t := stdcli.NewTable("ID", "STATUS", "RELEASE", "STARTED", "ELAPSED")

	for _, build := range builds {
		started := humanize.Time(build.Started)
		elapsed := stdcli.Duration(build.Started, build.Ended)

		if build.Ended.IsZero() {
			elapsed = ""
		}

		t.AddRow(build.Id, build.Status, build.Release, started, elapsed)
	}

	t.Print()
}

func cmdBuildsCreate(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	_, err = rackClient(c).GetApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	release, err := executeBuild(c, dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Release: %s\n", release)
}

func cmdBuildsInfo(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return
	}

	build := c.Args()[0]

	b, err := rackClient(c).GetBuild(app, build)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Println(b.Logs)
}

func executeBuild(c *cli.Context, dir string, app string) (string, error) {
	dir, err := filepath.Abs(dir)

	if err != nil {
		return "", err
	}

	fmt.Print("Uploading... ")

	tar, err := createTarball(dir)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	build, err := rackClient(c).CreateBuild(app, tar)

	if err != nil {
		return "", err
	}

	err = rackClient(c).StreamBuildLogs(app, build.Id, os.Stdout)

	if err != nil {
		return "", err
	}

	release, err := waitForBuild(c, app, build.Id)

	if err != nil {
		return "", err
	}

	return release, nil
}

func createTarball(base string) ([]byte, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(base)

	if err != nil {
		return nil, err
	}

	cmd := exec.Command("tar", "cz", ".")

	out, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	cmd.Start()

	bytes, err := ioutil.ReadAll(out)

	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(cwd)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func waitForBuild(c *cli.Context, app, id string) (string, error) {
	for {
		build, err := rackClient(c).GetBuild(app, id)

		if err != nil {
			return "", err
		}

		switch build.Status {
		case "complete":
			return build.Release, nil
		case "error":
			return "", fmt.Errorf("%s build failed", app)
		case "failed":
			return "", fmt.Errorf("%s build failed", app)
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("can't get here")
}
