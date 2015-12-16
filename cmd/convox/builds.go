package main

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "build",
		Description: "create a new build",
		Usage:       "",
		Action:      cmdBuildsCreate,
		Flags: []cli.Flag{
			appFlag,
			cli.BoolFlag{
				Name:  "no-cache",
				Usage: "Do not use Docker cache during build.",
			},
			cli.StringFlag{
				Name:  "file, f",
				Value: "docker-compose.yml",
				Usage: "a file to use in place of docker-compose.yml",
			},
		},
	})
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
				Flags: []cli.Flag{
					appFlag,
					cli.StringFlag{
						Name:  "file, f",
						Value: "docker-compose.yml",
						Usage: "a file to use in place of docker-compose.yml",
					},
				},
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
		started := humanizeTime(build.Started)
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

	a, err := rackClient(c).GetApp(app)

	if err != nil {
		stdcli.Error(err)
		return
	}

	switch a.Status {
	case "creating":
		stdcli.Error(fmt.Errorf("app is still creating: %s", app))
		return
	case "running", "updating":
	default:
		stdcli.Error(fmt.Errorf("unable to build app: %s", app))
		return
	}

	if len(c.Args()) > 0 {
		dir = c.Args()[0]
	}

	release, err := executeBuild(c, dir, app, c.String("file"))

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

func executeBuild(c *cli.Context, source string, app string, config string) (string, error) {
	u, _ := url.Parse(source)

	switch u.Scheme {
	case "http", "https":
		return executeBuildUrl(c, source, app, config)
	default:
		return executeBuildDir(c, source, app, config)
	}

	return "", fmt.Errorf("unreachable")
}

func executeBuildDir(c *cli.Context, dir string, app string, config string) (string, error) {
	dir, err := filepath.Abs(dir)

	if err != nil {
		return "", err
	}

	fmt.Print("Creating tarball... ")

	tar, err := createTarball(dir)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	cache := !c.Bool("no-cache")

	fmt.Print("Uploading... ")

	build, err := rackClient(c).CreateBuildSource(app, tar, cache, config)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	return finishBuild(c, app, build)
}

func executeBuildUrl(c *cli.Context, url string, app string, config string) (string, error) {
	cache := !c.Bool("no-cache")

	build, err := rackClient(c).CreateBuildUrl(app, url, cache, config)

	if err != nil {
		return "", err
	}

	return finishBuild(c, app, build)
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

	args := []string{"c"}

	// If .dockerignore exists, use it to exclude files from the tarball
	if _, err = os.Stat(".dockerignore"); err == nil {
		args = append(args, "--exclude-from", ".dockerignore")
	}

	args = append(args, ".")

	tar := exec.Command("tar", args...)
	gzip := exec.Command("gzip", "-f")

	r, w := io.Pipe()
	tar.Stdout = w
	gzip.Stdin = r

	var b bytes.Buffer
	gzip.Stdout = &b

	tar.Start()
	gzip.Start()
	tar.Wait()
	w.Close()
	gzip.Wait()

	err = os.Chdir(cwd)

	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func finishBuild(c *cli.Context, app string, build *client.Build) (string, error) {
	if build.Id == "" {
		return "", fmt.Errorf("unable to fetch build id")
	}

	reader, writer := io.Pipe()
	go io.Copy(os.Stdout, reader)
	err := rackClient(c).StreamBuildLogs(app, build.Id, writer)

	if err != nil {
		return "", err
	}

	release, err := waitForBuild(c, app, build.Id)

	if err != nil {
		return "", err
	}

	return release, nil
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
