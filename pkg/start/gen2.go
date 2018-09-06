package start

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/convox/changes"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
)

func (s *Start) Start2(p structs.Provider, opts Options) error {
	mf := helpers.Coalesce(opts.Manifest, "convox.yml")

	data, err := ioutil.ReadFile(mf)
	if err != nil {
		return err
	}

	app := opts.App

	if _, err := p.AppGet(app); err != nil {
		if _, err := p.AppCreate(app, structs.AppCreateOptions{Generation: options.String("2")}); err != nil {
			return err
		}
	}

	env, err := helpers.AppEnvironment(p, app)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return err
	}

	services := map[string]bool{}

	if opts.Services == nil {
		for _, s := range m.Services {
			services[s.Name] = true
		}
	} else {
		for _, s := range opts.Services {
			if _, err := m.Service(s); err != nil {
				return err
			}
			services[s] = true
		}
	}

	if opts.Build {
		m.Writef("build", "uploading source\n")

		data, err := helpers.Tarball(".")
		if err != nil {
			return err
		}

		o, err := p.ObjectStore(app, "", bytes.NewReader(data), structs.ObjectStoreOptions{})
		if err != nil {
			return err
		}

		m.Writef("build", "starting build\n")

		b, err := p.BuildCreate(app, o.Url, structs.BuildCreateOptions{Manifest: options.String(mf)})
		if err != nil {
			return err
		}

		logs, err := p.BuildLogs(app, b.Id, structs.LogsOptions{})
		if err != nil {
			return err
		}

		bo := m.Writer("build", os.Stdout)

		if _, err := io.Copy(bo, logs); err != nil {
			return err
		}

		if err := waitForBuild(p, app, b.Id); err != nil {
			return err
		}

		b, err = p.BuildGet(app, b.Id)
		if err != nil {
			return err
		}

		if err := p.ReleasePromote(app, b.Release); err != nil {
			return err
		}
	}

	errch := make(chan error)

	go handleInterrupt(func() {
		pss, err := p.ProcessList(app, structs.ProcessListOptions{})
		if err != nil {
			errch <- err
			return
		}

		var wg sync.WaitGroup

		wg.Add(len(pss))

		for _, ps := range pss {
			m.Writef("convox", "stopping %s\n", ps.Id)

			go func(id string) {
				defer wg.Done()
				p.ProcessStop(app, id)
			}(ps.Id)
		}

		wg.Wait()

		errch <- nil
	})

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, s := range m.Services {
		if !services[s.Name] {
			continue
		}

		if s.Build.Path != "" {
			go watchChanges(p, m, app, s.Name, wd, errch)
		}

		if s.Port.Port > 0 {
			go healthCheck(p, m, app, s, errch)
		}
	}

	go streamLogs(p, m, app, services)

	return <-errch
}

func healthCheck(p structs.Provider, m *manifest.Manifest, app string, s manifest.Service, errch chan error) {
	rss, err := p.ServiceList(app)
	if err != nil {
		errch <- err
		return
	}

	hostname := ""

	for _, rs := range rss {
		if rs.Name == s.Name {
			hostname = rs.Domain
		}
	}

	if hostname == "" {
		errch <- fmt.Errorf("could not find hostname for service: %s", s.Name)
		return
	}

	m.Writef("convox", "starting health check for <service>%s</service> on path <setting>%s</setting> with <setting>%d</setting>s interval, <setting>%d</setting>s grace\n", s.Name, s.Health.Path, s.Health.Interval, s.Health.Grace)

	hcu := fmt.Sprintf("https://%s%s", hostname, s.Health.Path)

	time.Sleep(time.Duration(s.Health.Grace) * time.Second)

	c := &http.Client{
		Timeout: time.Duration(s.Health.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for range time.Tick(time.Duration(s.Health.Interval) * time.Second) {
		res, err := c.Get(hcu)
		if err != nil {
			m.Writef("convox", "health check <service>%s</service>: <fail>%s</fail>\n", s.Name, err.Error())
			continue
		}

		if res.StatusCode < 200 || res.StatusCode > 399 {
			m.Writef("convox", "health check <service>%s</service>: <fail>%d</fail>\n", s.Name, res.StatusCode)
			continue
		}

		m.Writef("convox", "health check <service>%s</service>: <ok>%d</ok>\n", s.Name, res.StatusCode)
	}
}

var reAppLog = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2}) ([^/]+)/([^/]+)/([^ ]+) (.*)$`)

func streamLogs(p structs.Provider, m *manifest.Manifest, app string, services map[string]bool) {
	for {
		fmt.Printf("app logs: %s", app)
		logs, err := p.AppLogs(app, structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)})
		if err == nil {
			writeLogs(m, logs, services)
		}

		time.Sleep(1 * time.Second)
	}
}

func writeLogs(m *manifest.Manifest, r io.Reader, services map[string]bool) {
	ls := bufio.NewScanner(r)

	for ls.Scan() {
		match := reAppLog.FindStringSubmatch(ls.Text())

		if len(match) != 7 {
			continue
		}

		if !services[match[4]] {
			continue
		}

		m.Writef(match[4], "%s\n", match[6])
	}
}

func waitForBuild(p structs.Provider, app, id string) error {
	for {
		time.Sleep(1 * time.Second)

		b, err := p.BuildGet(app, id)
		if err != nil {
			return err
		}

		switch b.Status {
		case "created", "running":
			break
		case "complete":
			return nil
		case "failed":
			return fmt.Errorf("build failed")
		default:
			return fmt.Errorf("unknown build status: %s", b.Status)
		}
	}
}

func watchChanges(p structs.Provider, m *manifest.Manifest, app, service, root string, ch chan error) {
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
		go watchPath(p, m, app, service, root, bs, ignores, ch)
	}
}

func watchPath(p structs.Provider, m *manifest.Manifest, app, service, root string, bs manifest.BuildSource, ignores []string, ch chan error) {
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

			pss, err := p.ProcessList(app, structs.ProcessListOptions{Service: options.String(service)})
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

				if err := handleAdds(p, app, ps.Id, bs.Remote, adds); err != nil {
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

				if err := handleRemoves(p, app, ps.Id, removes); err != nil {
					w.Writef("sync remove error: %s\n", err)
				}
			}

			chgs = []changes.Change{}
		}
	}
}

func handleAdds(p structs.Provider, app, pid, remote string, adds []changes.Change) error {
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
		ch <- p.FilesUpload(app, pid, rp)
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

func handleRemoves(p structs.Provider, app, pid string, removes []changes.Change) error {
	if len(removes) == 0 {
		return nil
	}

	return p.FilesDelete(app, pid, changes.Files(removes))
}
