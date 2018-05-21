package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "releases",
		Description: "list an app's releases",
		Usage:       "[subcommand] [options]",
		ArgsUsage:   "[subcommand]",
		Action:      cmdReleases,
		Flags: []cli.Flag{
			appFlag,
			rackFlag,
			cli.IntFlag{
				Name:  "limit",
				Usage: "number of releases to list",
				Value: 20,
			},
		},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new release",
				Usage:       "",
				Action:      cmdReleaseCreate,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
					cli.StringFlag{
						Name:  "build",
						Usage: "build id",
					},
					cli.StringFlag{
						Name:  "manifest",
						Usage: "manifest file",
					},
					cli.BoolFlag{
						Name:  "promote",
						Usage: "promote the release after env change",
					},
					cli.BoolFlag{
						Name:   "wait",
						EnvVar: "CONVOX_WAIT",
						Usage:  "wait for release to finish promoting before returning",
					},
				},
			},
			{
				Name:        "info",
				Description: "see info about a release",
				Usage:       "<release id>",
				Action:      cmdReleaseInfo,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "promote",
				Description: "promote a release",
				Usage:       "[release] [options]",
				ArgsUsage:   "[release]",
				Action:      cmdReleasePromote,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
					cli.BoolFlag{
						Name:   "wait",
						EnvVar: "CONVOX_WAIT",
						Usage:  "wait for release to finish promoting before returning",
					},
				},
			},
			{
				Name:        "rollback",
				Description: "create a new release from an old release and promote it",
				Usage:       "<release> [options]",
				ArgsUsage:   "<release>",
				Action:      cmdReleaseRollback,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
					cli.BoolFlag{
						Name:   "wait",
						EnvVar: "CONVOX_WAIT",
						Usage:  "wait for release to finish promoting before returning",
					},
				},
			},
		},
	})
}

func cmdReleases(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	var releases client.Releases
	if c.IsSet("limit") {
		releases, err = rackClient(c).GetReleasesWithLimit(app, c.Int("limit"))
	} else {
		releases, err = rackClient(c).GetReleases(app)
	}
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("ID", "CREATED", "BUILD", "STATUS")

	for _, r := range releases {
		status := ""

		if a.Release == r.Id {
			status = "active"
		}

		t.AddRow(r.Id, helpers.HumanizeTime(r.Created), r.Build, status)
	}

	t.Print()
	return nil
}

func cmdReleaseCreate(c *cli.Context) error {
	stdcli.NeedHelp(c)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	opts := structs.ReleaseCreateOptions{}

	if v := c.String("build"); v != "" {
		opts.Build = options.String(v)
	}

	if v := c.String("manifest"); v != "" {
		data, err := ioutil.ReadFile(v)
		if err != nil {
			return stdcli.Error(err)
		}

		opts.Manifest = options.String(string(data))
	}

	stdcli.Startf("Creating release")
	r, err := rack(c).ReleaseCreate(app, opts)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Writef("<ok>OK</ok>, <release>%s</release>\n", r.Id)

	if c.Bool("promote") {
		stdcli.Startf("Promoting <release>%s</release>", r.Id)

		if err := rack(c).ReleasePromote(app, r.Id); err != nil {
			return stdcli.Error(err)
		}

		stdcli.OK()

		if c.Bool("wait") {
			if err := waitForReleasePromotion(os.Stdout, c, app, r.Id); err != nil {
				return stdcli.Error(err)
			}
		}
	}

	return nil
}

func cmdReleaseInfo(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	release := c.Args()[0]

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	r, err := rackClient(c).GetRelease(app, release)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Id       %s\n", r.Id)
	fmt.Printf("Build    %s\n", r.Build)
	fmt.Printf("Created  %s\n", r.Created)
	fmt.Printf("Env      ")

	fmt.Println(strings.Replace(r.Env, "\n", "\n         ", -1))
	return nil
}

func cmdReleasePromote(c *cli.Context) error {
	stdcli.NeedHelp(c)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	var release string

	if len(c.Args()) > 0 {
		release = c.Args()[0]
	} else {
		rr, err := rackClient(c).GetReleases(app)
		if err != nil {
			return stdcli.Error(err)
		}

		if len(rr) < 1 {
			return stdcli.Error(fmt.Errorf("no releases for app: %s", app))
		}

		release = rr[0].Id
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	if a.Status != "running" {
		return stdcli.Error(fmt.Errorf("app %s is still being updated, check `convox apps info`", app))
	}

	fmt.Printf("Promoting %s... ", release)

	_, err = rackClient(c).PromoteRelease(app, release)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Println("UPDATING")

	if c.Bool("wait") {
		if err := waitForReleasePromotion(os.Stdout, c, app, release); err != nil {
			return stdcli.Error(err)
		}
	}

	return nil
}

func cmdReleaseRollback(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	if a.Status != "running" {
		return stdcli.Error(fmt.Errorf("%s is currently being updated", app))
	}

	id := c.Args()[0]

	ro, err := rack(c).ReleaseGet(app, id)
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Startf("Copying release <release>%s</release>", ro.Id)

	rn, err := rack(c).ReleaseCreate(app, structs.ReleaseCreateOptions{
		Build: options.String(ro.Build),
		Env:   options.String(ro.Env),
	})
	if err != nil {
		return stdcli.Error(err)
	}

	stdcli.Writef("<ok>OK</ok>, <release>%s</release>\n", rn.Id)

	stdcli.Startf("Promoting <release>%s</release>", rn.Id)

	if err := rack(c).ReleasePromote(app, rn.Id); err != nil {
		return stdcli.Error(err)
	}

	stdcli.OK()

	if c.Bool("wait") {
		if err := waitForReleasePromotion(os.Stdout, c, app, rn.Id); err != nil {
			return stdcli.Error(err)
		}
	}

	return nil
}

func waitForReleasePromotion(out io.Writer, c *cli.Context, app, release string) error {
	out.Write([]byte("Waiting for stabilization...\n"))

	done := make(chan bool)

	go streamAppSystemLogs(out, c, app, done)

	if err := waitForAppRunning(c, app); err != nil {
		return err
	}

	done <- true

	out.Write([]byte("OK\n"))

	return nil
}

func streamAppSystemLogs(out io.Writer, c *cli.Context, app string, done chan bool) {
	r, w := io.Pipe()

	defer r.Close()

	go rackClient(c).StreamAppLogs(app, "", true, 0*time.Second, w)
	go copySystemLogs(out, r)

	<-done
}

func copySystemLogs(w io.Writer, r io.Reader) {
	s := bufio.NewScanner(r)

	for s.Scan() {
		parts := strings.SplitN(s.Text(), " ", 3)

		if len(parts) < 3 {
			continue
		}

		if strings.HasPrefix(parts[1], "system/aws") {
			w.Write([]byte(fmt.Sprintf("%s\n", s.Text())))
		}
	}
}
