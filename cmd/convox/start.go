package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/convox/changes"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/options"
	"github.com/convox/rack/sdk"
	"github.com/convox/rack/structs"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/urfave/cli.v1"
)

var (
	reAppLog = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2}) ([^/]+)/([^/]+)/([^ ]+) (.*)$`)
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "start",
		Description: "start an app for local development",
		Usage:       "[service] [command]",
		Action:      cmdStart,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "file, f",
				EnvVar: "COMPOSE_FILE",
				Value:  "",
				Usage:  "path to manifest file",
			},
			cli.StringFlag{
				Name:  "generation, g",
				Usage: "generation of app",
			},
			cli.BoolFlag{
				Name:  "no-build",
				Usage: "dont run the build process",
			},
			cli.BoolFlag{
				Name:  "no-cache",
				Usage: "pull fresh image dependencies",
			},
			cli.BoolFlag{
				Name:  "no-sync",
				Usage: "do not synchronize local file changes into the running containers",
			},
			cli.IntFlag{
				Name:  "shift",
				Usage: "shift allocated port numbers by the given amount",
			},
		},
	})
}

func cmdStart(c *cli.Context) error {
	if err := dockerTest(); err != nil {
		return stdcli.Error(err)
	}

	opts := startOptions{}

	if len(c.Args()) > 0 {
		opts.Service = c.Args()[0]
	}

	if len(c.Args()) > 1 {
		opts.Command = c.Args()[1:]
	}

	opts.Build = !c.Bool("no-build")
	opts.Cache = !c.Bool("no-cache")
	opts.Sync = !c.Bool("no-sync")

	if v := c.String("file"); v != "" {
		opts.Config = v
	}

	if v := c.Int("shift"); v > 0 {
		opts.Shift = v
	}

	opts.Id, _ = currentId()

	if stdcli.ReadSetting("generation") == "2" || c.String("generation") == "2" || filepath.Base(opts.Config) == "convox.yml" {
		if err := startGeneration2(opts); err != nil {
			return stdcli.Error(err)
		}
	} else {
		if err := startGeneration1(opts); err != nil {
			return stdcli.Error(err)
		}
	}

	return nil
}

type startOptions struct {
	Build   bool
	Cache   bool
	Command []string
	Config  string
	Id      string
	Service string
	Shift   int
	Sync    bool
}

func startGeneration1(opts startOptions) error {
	opts.Config = helpers.Coalesce(opts.Config, "docker-compose.yml")

	if !helpers.Exists(opts.Config) {
		return fmt.Errorf("manifest not found: %s", opts.Config)
	}

	app, err := stdcli.DefaultApp(opts.Config)
	if err != nil {
		return err
	}

	m, err := manifest1.LoadFile(opts.Config)
	if err != nil {
		return err
	}

	if errs := m.Validate(); len(errs) > 0 {
		for _, e := range errs[1:] {
			stdcli.Error(e)
		}
		return errs[0]
	}

	if s := opts.Service; s != "" {
		if _, ok := m.Services[s]; !ok {
			return fmt.Errorf("service %s not found in manifest", s)
		}
	}

	if err := m.Shift(opts.Shift); err != nil {
		return err
	}

	// one-off commands don't need port validation
	if len(opts.Command) == 0 {
		pcc, err := m.PortConflicts()
		if err != nil {
			return err
		}
		if len(pcc) > 0 {
			return fmt.Errorf("ports in use: %v", pcc)
		}
	}

	r := m.Run(filepath.Dir(opts.Config), app, manifest1.RunOptions{
		Build:   opts.Build,
		Cache:   opts.Cache,
		Command: opts.Command,
		Service: opts.Service,
		Sync:    opts.Sync,
	})

	err = r.Start()
	if err != nil {
		r.Stop()
		return err
	}

	go handleInterrupt1(r)

	return r.Wait()
}

func startGeneration2(opts startOptions) error {
	if !localRackRunning() {
		return fmt.Errorf("local rack not found, try `sudo convox rack install local`")
	}

	data, err := ioutil.ReadFile("convox.yml")
	if err != nil {
		return err
	}

	app, err := stdcli.DefaultApp(opts.Config)
	if err != nil {
		return err
	}

	rack, err := sdk.New("https://localhost:5443")
	if err != nil {
		return err
	}

	if _, err := rack.AppGet(app); err != nil {
		if _, err := rack.AppCreate(app, structs.AppCreateOptions{Generation: options.String("2")}); err != nil {
			return err
		}
	}

	env, err := rack.EnvironmentGet(app)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data, manifest.Environment(env))
	if err != nil {
		return err
	}

	tar, err := createTarball(".")
	if err != nil {
		return err
	}

	o, err := rack.ObjectStore(app, "", bytes.NewReader(tar), structs.ObjectStoreOptions{})
	if err != nil {
		return err
	}

	b, err := rack.BuildCreate(app, "tgz", o.Url, structs.BuildCreateOptions{})
	if err != nil {
		return err
	}

	if err := waitForBuildGeneration2(rack, app, b.Id); err != nil {
		return err
	}

	logs, err := rack.BuildLogs(app, b.Id, structs.LogsOptions{})
	if err != nil {
		return err
	}

	bo := m.Writer("build", os.Stdout)

	if _, err := io.Copy(bo, logs); err != nil {
		return err
	}

	b, err = rack.BuildGet(app, b.Id)
	if err != nil {
		return err
	}

	switch b.Status {
	case "created", "running", "complete":
	case "failed":
		return fmt.Errorf("build failed")
	default:
		return fmt.Errorf("unknown build status: %s", b.Status)
	}

	if err := rack.ReleasePromote(app, b.Release); err != nil {
		return err
	}

	r, err := rack.ReleaseGet(app, b.Release)
	if err != nil {
		return err
	}

	switch r.Status {
	case "created", "promoting", "promoted", "active":
	case "failed":
		return fmt.Errorf("release failed")
	default:
		return fmt.Errorf("unknown release status: %s", r.Status)
	}

	errch := make(chan error)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go handleSignals(rack, m, app, sig, errch)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, s := range m.Services {
		if s.Build.Path != "" {
			go watchChanges(rack, m, app, s.Name, wd, errch)
		}
	}

	logs, err = rack.AppLogs(app, structs.LogsOptions{Follow: true, Prefix: true})
	if err != nil {
		return err
	}

	ls := bufio.NewScanner(logs)

	go func() {
		for ls.Scan() {
			match := reAppLog.FindStringSubmatch(ls.Text())

			if len(match) == 7 {
				m.Writef(match[4], "%s\n", match[6])
			}
		}
	}()

	// if _, err := io.Copy(os.Stdout, logs); err != nil {
	//   return err
	// }

	return <-errch
}

// func startGeneration2(opts startOptions) error {
//   opts.Config = helpers.Coalesce(opts.Config, "convox.yml")

//   if !helpers.Exists(opts.Config) {
//     return fmt.Errorf("manifest not found: %s", opts.Config)
//   }

//   app, err := stdcli.DefaultApp(opts.Config)
//   if err != nil {
//     return err
//   }

//   data, err := ioutil.ReadFile(opts.Config)
//   if err != nil {
//     return err
//   }

//   dir := filepath.Dir(app)

//   env := structs.Environment{}

//   if data, err := ioutil.ReadFile(filepath.Join(dir, ".env")); err == nil {
//     if err := env.Load(data); err != nil {
//       return err
//     }
//   }

//   m, err := manifest.Load(data, manifest.Environment(env))
//   if err != nil {
//     return err
//   }

//   bopts := manifest.BuildOptions{
//     Development: true,
//     Stdout:      m.Writer("build", os.Stdout),
//     Stderr:      m.Writer("build", os.Stderr),
//   }

//   if err := m.Build(app, "latest", bopts); err != nil {
//     return err
//   }

//   ropts := manifest.RunOptions{
//     Bind:   true,
//     Env:    env,
//     Stdout: os.Stdout,
//     Stderr: os.Stderr,
//   }

//   if err := m.Run(app, ropts); err != nil {
//     return err
//   }

//   return nil
// }

func handleInterrupt1(run manifest1.Run) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	run.Stop()
	os.Exit(0)
}

func handleSignals(rack *sdk.Client, m *manifest.Manifest, app string, ch chan os.Signal, errch chan error) {
	sig := <-ch

	if sig == syscall.SIGINT {
		fmt.Println("")
	}

	ps, err := rack.ProcessList(app, structs.ProcessListOptions{})
	if err != nil {
		errch <- err
		return
	}

	var wg sync.WaitGroup

	wg.Add(len(ps))

	for _, p := range ps {
		m.Writef("convox", "stopping %s\n", p.Id)

		go func(id string) {
			defer wg.Done()
			rack.ProcessStop(app, id)
		}(p.Id)
	}

	wg.Wait()

	os.Exit(0)
}

func dockerTest() error {
	dockerTest := exec.Command("docker", "images")
	err := dockerTest.Run()
	if err != nil {
		return errors.New("could not connect to docker daemon, is it installed and running?")
	}

	dockerVersionTest, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	minDockerVersion, err := docker.NewAPIVersion("1.9")
	if err != nil {
		return err
	}
	e, err := dockerVersionTest.Version()
	if err != nil {
		return err
	}

	currentVersionParts := strings.Split(e.Get("Version"), ".")
	currentVersion, err := docker.NewAPIVersion(fmt.Sprintf("%s.%s", currentVersionParts[0], currentVersionParts[1]))
	if err != nil {
		return err
	}

	if !(currentVersion.GreaterThanOrEqualTo(minDockerVersion)) {
		return errors.New("Your version of docker is out of date (min: 1.9)")
	}
	return nil
}

func waitForBuildGeneration2(rack *sdk.Client, app, id string) error {
	for {
		time.Sleep(1 * time.Second)

		b, err := rack.BuildGet(app, id)
		if err != nil {
			return err
		}

		switch b.Status {
		case "created":
			break
		case "running", "complete":
			return nil
		case "failed":
			return fmt.Errorf("build failed")
		default:
			return fmt.Errorf("unknown build status: %s", b.Status)
		}
	}
}

func watchChanges(rack *sdk.Client, m *manifest.Manifest, app, service, root string, ch chan error) {
	bss, err := m.BuildSources(root, service)
	if err != nil {
		ch <- err
		return
	}

	ignores, err := m.BuildIgnores(root, service)
	if err != nil {
		ch <- err
		return
	}

	for _, bs := range bss {
		go watchPath(rack, m, app, service, root, bs, ignores, ch)
	}
}

func watchPath(rack *sdk.Client, m *manifest.Manifest, app, service, root string, bs manifest.BuildSource, ignores []string, ch chan error) {
	cch := make(chan changes.Change, 1)

	w := m.Writer("convox", os.Stdout)

	abs, err := filepath.Abs(bs.Local)
	if err != nil {
		ch <- err
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		ch <- err
		return
	}

	rel, err := filepath.Rel(wd, bs.Local)
	if err != nil {
		ch <- err
		return
	}

	m.Writef("convox", "syncing: <dir>%s</dir> to <dir>%s:%s</dir>\n", rel, service, bs.Remote)

	go changes.Watch(abs, cch, changes.WatchOptions{
		Ignores: ignores,
	})

	tick := time.Tick(1000 * time.Millisecond)
	chgs := []changes.Change{}

	for {
		select {
		case c := <-cch:
			chgs = append(chgs, c)
		case <-tick:
			if len(chgs) == 0 {
				continue
			}

			pss, err := rack.ProcessList(app, structs.ProcessListOptions{Service: options.String(service)})
			if err != nil {
				w.Writef("sync error: %s\n", err)
				continue
			}

			adds, removes := changes.Partition(chgs)

			for _, ps := range pss {
				switch {
				case len(adds) > 3:
					w.Writef("sync: %d files to <dir>%s:%s</dir>\n", len(adds), service, bs.Remote)
				case len(adds) > 0:
					for _, a := range adds {
						w.Writef("sync: <dir>%s</dir> to <dir>%s:%s</dir>\n", a.Path, service, bs.Remote)
					}
				}

				if err := handleAdds(rack, app, ps.Id, bs.Remote, adds); err != nil {
					w.Writef("sync add error: %s\n", err)
				}

				switch {
				case len(removes) > 3:
					w.Writef("remove: %d files from <dir>%s:%s</dir>\n", len(removes), service, bs.Remote)
				case len(removes) > 0:
					for _, r := range removes {
						w.Writef("remove: <dir>%s</dir> from <dir>%s:%s</dir>\n", r.Path, service, bs.Remote)
					}
				}

				if err := handleRemoves(rack, app, ps.Id, removes); err != nil {
					w.Writef("sync remove error: %s\n", err)
				}
			}

			chgs = []changes.Change{}
		}
	}
}

func handleAdds(rack *sdk.Client, app, pid, remote string, adds []changes.Change) error {
	if len(adds) == 0 {
		return nil
	}

	if !filepath.IsAbs(remote) {
		data, err := exec.Command("docker", "inspect", pid, "--format", "{{.Config.WorkingDir}}").CombinedOutput()
		if err != nil {
			return fmt.Errorf("container inspect %s %s", string(data), err)
		}

		wd := strings.TrimSpace(string(data))

		remote = filepath.Join(wd, remote)
	}

	rp, wp := io.Pipe()

	ch := make(chan error)

	go func() {
		ch <- rack.FilesUpload(app, pid, rp)
	}()

	tgz := gzip.NewWriter(wp)
	tw := tar.NewWriter(tgz)

	for _, add := range adds {
		local := filepath.Join(add.Base, add.Path)

		stat, err := os.Stat(local)
		if err != nil {
			// skip transient files like '.git/.COMMIT_EDITMSG.swp'
			if os.IsNotExist(err) {
				continue
			}

			return err
		}

		tw.WriteHeader(&tar.Header{
			Name:    filepath.Join(remote, add.Path),
			Mode:    int64(stat.Mode()),
			Size:    stat.Size(),
			ModTime: stat.ModTime(),
		})

		fd, err := os.Open(local)
		if err != nil {
			return err
		}

		defer fd.Close()

		if _, err := io.Copy(tw, fd); err != nil {
			return err
		}

		fd.Close()
	}

	if err := tw.Close(); err != nil {
		return err
	}

	if err := tgz.Close(); err != nil {
		return err
	}

	if err := wp.Close(); err != nil {
		return err
	}

	return <-ch
}

func handleRemoves(rack *sdk.Client, app, pid string, removes []changes.Change) error {
	if len(removes) == 0 {
		return nil
	}

	return rack.FilesDelete(app, pid, changes.Files(removes))
}
