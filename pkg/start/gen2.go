package start

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/convox/changes"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/prefix"
	"github.com/convox/rack/pkg/structs"
	"github.com/docker/docker/builder/dockerignore"
)

const (
	ScannerStartSize = 4096
	ScannerMaxSize   = 1024 * 1024
)

var (
	reAppLog = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2}) ([^/]+)/([^/]+)/([^ ]+) (.*)$`)
)

type Options2 struct {
	App      string
	Build    bool
	Cache    bool
	Manifest string
	Provider structs.Provider
	Services []string
	Sync     bool
	Test     bool
}

type buildSource struct {
	Local  string
	Remote string
}

func (s *Start) Start2(ctx context.Context, w io.Writer, opts Options2) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	if opts.App == "" {
		return fmt.Errorf("app required")
	}

	a, err := opts.Provider.AppGet(opts.App)
	if err != nil {
		if _, err := opts.Provider.AppCreate(opts.App, structs.AppCreateOptions{Generation: options.String("2")}); err != nil {
			return err
		}
	} else {
		if a.Generation != "2" {
			return fmt.Errorf("invalid generation: %s", a.Generation)
		}
	}

	data, err := ioutil.ReadFile(helpers.Coalesce(opts.Manifest, "convox.yml"))
	if err != nil {
		return err
	}

	env, err := helpers.AppEnvironment(opts.Provider, opts.App)
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
			services[s] = true
		}
	}

	pw := prefixWriter(w, services)

	if opts.Build {
		pw.Writef("build", "uploading source\n")

		data, err := helpers.Tarball(".")
		if err != nil {
			return err
		}

		o, err := opts.Provider.ObjectStore(opts.App, "", bytes.NewReader(data), structs.ObjectStoreOptions{})
		if err != nil {
			return err
		}

		pw.Writef("build", "starting build\n")

		bopts := structs.BuildCreateOptions{Development: options.Bool(true)}

		if opts.Manifest != "" {
			bopts.Manifest = options.String(opts.Manifest)
		}

		b, err := opts.Provider.BuildCreate(opts.App, o.Url, bopts)
		if err != nil {
			return err
		}

		logs, err := opts.Provider.BuildLogs(opts.App, b.Id, structs.LogsOptions{})
		if err != nil {
			return err
		}

		bo := pw.Writer("build")

		go io.Copy(bo, logs)

		if err := opts.waitForBuild(ctx, b.Id); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		b, err = opts.Provider.BuildGet(opts.App, b.Id)
		if err != nil {
			return err
		}

		if err := opts.Provider.ReleasePromote(opts.App, b.Release); err != nil {
			return err
		}
	}

	errch := make(chan error)
	defer close(errch)

	go handleErrors(ctx, pw, errch)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, s := range m.Services {
		if !services[s.Name] {
			continue
		}

		if s.Build.Path != "" {
			go opts.watchChanges(ctx, pw, m, s.Name, wd, errch)
		}

		if s.Port.Port > 0 {
			wg.Add(1)
			go opts.healthCheck(ctx, pw, s, errch, &wg)
		}
	}

	wg.Wait()

	go opts.streamLogs(ctx, pw, services)

	<-ctx.Done()

	pss, err := opts.Provider.ProcessList(opts.App, structs.ProcessListOptions{})
	if err != nil {
		return nil
	}

	wg.Add(len(pss))

	for _, ps := range pss {
		pw.Writef("convox", "stopping %s\n", ps.Id)
		go opts.stopProcess(ps.Id, &wg)
	}

	wg.Wait()

	return nil
}

func (opts Options2) handleAdds(pid, remote string, adds []changes.Change) error {
	if len(adds) == 0 {
		return nil
	}

	if !filepath.IsAbs(remote) {
		data, err := Exec.Execute("docker", "inspect", pid, "--format", "{{.Config.WorkingDir}}")
		if err != nil {
			return fmt.Errorf("container inspect %s %s", string(data), err)
		}

		wd := strings.TrimSpace(string(data))

		remote = filepath.Join(wd, remote)
	}

	rp, wp := io.Pipe()

	ch := make(chan error)
	defer close(ch)

	go func() {
		ch <- opts.Provider.FilesUpload(opts.App, pid, rp)
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

func (opts Options2) handleRemoves(pid string, removes []changes.Change) error {
	if len(removes) == 0 {
		return nil
	}

	return opts.Provider.FilesDelete(opts.App, pid, changes.Files(removes))
}

func (opts Options2) healthCheck(ctx context.Context, pw prefix.Writer, s manifest.Service, errch chan error, wg *sync.WaitGroup) {
	rss, err := opts.Provider.ServiceList(opts.App)
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

	pw.Writef("convox", "starting health check for <service>%s</service> on path <setting>%s</setting> with <setting>%d</setting>s interval, <setting>%d</setting>s grace\n", s.Name, s.Health.Path, s.Health.Interval, s.Health.Grace)

	wg.Done()

	hcu := fmt.Sprintf("https://%s%s", hostname, s.Health.Path)

	grace := time.Duration(s.Health.Grace) * time.Second
	interval := time.Duration(s.Health.Interval) * time.Second

	if opts.Test {
		grace = 5 * time.Millisecond
		interval = 5 * time.Millisecond
	}

	time.Sleep(grace)

	tick := time.Tick(interval)

	c := &http.Client{
		Timeout: time.Duration(s.Health.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// previous status code
	var ps int

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			res, err := c.Get(hcu)
			if err != nil {
				pw.Writef("convox", "health check <service>%s</service>: <fail>%s</fail>\n", s.Name, err.Error())
				continue
			}
			if res.StatusCode < 200 || res.StatusCode > 399 {
				pw.Writef("convox", "health check <service>%s</service>: <fail>%d</fail>\n", s.Name, res.StatusCode)
			} else if res.StatusCode != ps {
				pw.Writef("convox", "health check <service>%s</service>: <ok>%d</ok>\n", s.Name, res.StatusCode)
			}
			ps = res.StatusCode
		}
	}
}

func (opts Options2) stopProcess(pid string, wg *sync.WaitGroup) {
	defer wg.Done()
	opts.Provider.ProcessStop(opts.App, pid)
}

func (opts Options2) streamLogs(ctx context.Context, pw prefix.Writer, services map[string]bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			logs, err := opts.Provider.AppLogs(opts.App, structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)})
			if err == nil {
				writeLogs(ctx, pw, logs, services)
			}

			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (opts Options2) waitForBuild(ctx context.Context, id string) error {
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick:
			b, err := opts.Provider.BuildGet(opts.App, id)
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
}

func (opts Options2) watchChanges(ctx context.Context, pw prefix.Writer, m *manifest.Manifest, service, root string, ch chan error) {
	bss, err := buildSources(m, root, service)
	if err != nil {
		ch <- err
		return
	}

	ignores, err := buildIgnores(root, service)
	if err != nil {
		ch <- err
		return
	}

	for _, bs := range bss {
		go opts.watchPath(ctx, pw, service, root, bs, ignores, ch)
	}
}

func (opts Options2) watchPath(ctx context.Context, pw prefix.Writer, service, root string, bs buildSource, ignores []string, ch chan error) {
	cch := make(chan changes.Change, 1)
	defer close(cch)

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

	pw.Writef("convox", "syncing: <dir>%s</dir> to <dir>%s:%s</dir>\n", rel, service, bs.Remote)

	go changes.Watch(abs, cch, changes.WatchOptions{
		Ignores: ignores,
	})

	tick := time.Tick(1000 * time.Millisecond)
	chgs := []changes.Change{}

	for {
		select {
		case <-ctx.Done():
			return
		case c := <-cch:
			chgs = append(chgs, c)
		case <-tick:
			if len(chgs) == 0 {
				continue
			}

			pss, err := opts.Provider.ProcessList(opts.App, structs.ProcessListOptions{Service: options.String(service)})
			if err != nil {
				pw.Writef("convox", "sync error: %s\n", err)
				continue
			}

			adds, removes := changes.Partition(chgs)

			for _, ps := range pss {
				switch {
				case len(adds) > 3:
					pw.Writef("convox", "sync: %d files to <dir>%s:%s</dir>\n", len(adds), service, bs.Remote)
				case len(adds) > 0:
					for _, a := range adds {
						pw.Writef("convox", "sync: <dir>%s</dir> to <dir>%s:%s</dir>\n", a.Path, service, bs.Remote)
					}
				}

				if err := opts.handleAdds(ps.Id, bs.Remote, adds); err != nil {
					pw.Writef("convox", "sync add error: %s\n", err)
				}

				switch {
				case len(removes) > 3:
					pw.Writef("convox", "remove: %d files from <dir>%s:%s</dir>\n", len(removes), service, bs.Remote)
				case len(removes) > 0:
					for _, r := range removes {
						pw.Writef("convox", "remove: <dir>%s</dir> from <dir>%s:%s</dir>\n", r.Path, service, bs.Remote)
					}
				}

				if err := opts.handleRemoves(ps.Id, removes); err != nil {
					pw.Writef("convox", "sync remove error: %s\n", err)
				}
			}

			chgs = []changes.Change{}
		}
	}
}

func buildDockerfile(m *manifest.Manifest, root, service string) ([]byte, error) {
	s, err := m.Service(service)
	if err != nil {
		return nil, err
	}

	if s.Image != "" {
		return nil, nil
	}

	path, err := filepath.Abs(filepath.Join(root, s.Build.Path, s.Build.Manifest))
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("no such file: %s", filepath.Join(s.Build.Path, s.Build.Manifest))
	}

	return ioutil.ReadFile(path)
}

func buildIgnores(root, service string) ([]string, error) {
	fd, err := os.Open(filepath.Join(root, ".dockerignore"))
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return dockerignore.ReadAll(fd)
}

func buildSources(m *manifest.Manifest, root, service string) ([]buildSource, error) {
	data, err := buildDockerfile(m, root, service)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return []buildSource{}, nil
	}

	svc, err := m.Service(service)
	if err != nil {
		return nil, err
	}

	bs := []buildSource{}
	env := map[string]string{}
	wd := ""

	s := bufio.NewScanner(bytes.NewReader(data))

	for s.Scan() {
		parts := strings.Fields(s.Text())

		if len(parts) < 1 {
			continue
		}

		switch strings.ToUpper(parts[0]) {
		case "ADD", "COPY":
			if len(parts) > 2 {
				u, err := url.Parse(parts[1])
				if err != nil {
					return nil, err
				}

				if strings.HasPrefix(parts[1], "--from") {
					continue
				}

				switch u.Scheme {
				case "http", "https":
					// do nothing
				default:
					local := filepath.Join(svc.Build.Path, parts[1])
					remote := replaceEnv(parts[2], env)

					if wd != "" && !filepath.IsAbs(remote) {
						remote = filepath.Join(wd, remote)
					}

					bs = append(bs, buildSource{Local: local, Remote: remote})
				}
			}
		case "ENV":
			if len(parts) > 2 {
				env[parts[1]] = parts[2]
			}
		case "FROM":
			if len(parts) > 1 {
				var ee []string

				data, err := Exec.Execute("docker", "inspect", parts[1], "--format", "{{json .Config.Env}}")
				if err != nil {
					return nil, err
				}

				if err := json.Unmarshal(data, &ee); err != nil {
					return nil, err
				}

				for _, e := range ee {
					parts := strings.SplitN(e, "=", 2)

					if len(parts) == 2 {
						env[parts[0]] = parts[1]
					}
				}

				data, err = Exec.Execute("docker", "inspect", parts[1], "--format", "{{.Config.WorkingDir}}")
				if err != nil {
					return nil, err
				}

				wd = strings.TrimSpace(string(data))
			}
		case "WORKDIR":
			if len(parts) > 1 {
				wd = replaceEnv(parts[1], env)
			}
		}
	}

	for i := range bs {
		abs, err := filepath.Abs(bs[i].Local)
		if err != nil {
			return nil, err
		}

		stat, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}

		if stat.IsDir() && !strings.HasSuffix(abs, "/") {
			abs = abs + "/"
		}

		bs[i].Local = abs

		if bs[i].Remote == "." {
			bs[i].Remote = wd
		}
	}

	bss := []buildSource{}

	for i := range bs {
		contained := false

		for j := i + 1; j < len(bs); j++ {
			if strings.HasPrefix(bs[i].Local, bs[j].Local) {
				if bs[i].Remote == bs[j].Remote {
					contained = true
					break
				}

				rl, err := filepath.Rel(bs[j].Local, bs[i].Local)
				if err != nil {
					return nil, err
				}

				rr, err := filepath.Rel(bs[j].Remote, bs[i].Remote)
				if err != nil {
					return nil, err
				}

				if rl == rr {
					contained = true
					break
				}
			}
		}

		if !contained {
			bss = append(bss, bs[i])
		}
	}

	return bss, nil
}

func handleErrors(ctx context.Context, pw prefix.Writer, errch chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errch:
			pw.Writef("convox", "<error>error: %s</error>\n", err)
		}
	}
}

func replaceEnv(s string, env map[string]string) string {
	for k, v := range env {
		s = strings.Replace(s, fmt.Sprintf("${%s}", k), v, -1)
		s = strings.Replace(s, fmt.Sprintf("$%s", k), v, -1)
	}

	return s
}

func writeLogs(ctx context.Context, pw prefix.Writer, r io.Reader, services map[string]bool) {
	ls := bufio.NewScanner(r)

	ls.Buffer(make([]byte, ScannerStartSize), ScannerMaxSize)

	for ls.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			match := reAppLog.FindStringSubmatch(ls.Text())

			if len(match) != 7 {
				continue
			}

			if !services[match[4]] {
				continue
			}

			pw.Writef(match[4], "%s\n", match[6])
		}
	}

	if err := ls.Err(); err != nil {
		pw.Writef("convox", "scan error: %s\n", err)
	}
}
