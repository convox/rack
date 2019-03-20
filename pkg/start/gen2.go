package start

import (
	"archive/tar"
	"bufio"
	"bytes"
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

	"github.com/pkg/errors"

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
	reAppLog       = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})Z ([^/]+)/([^/]+)/([^ ]+) (.*)$`)
	reDockerOption = regexp.MustCompile("--([a-z]+)")
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
		return errors.WithStack(fmt.Errorf("app required"))
	}

	a, err := opts.Provider.AppGet(opts.App)
	if err != nil {
		if _, err := opts.Provider.AppCreate(opts.App, structs.AppCreateOptions{Generation: options.String("2")}); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if a.Generation != "2" {
			return errors.WithStack(fmt.Errorf("invalid generation: %s", a.Generation))
		}
	}

	data, err := ioutil.ReadFile(helpers.CoalesceString(opts.Manifest, "convox.yml"))
	if err != nil {
		return errors.WithStack(err)
	}

	env, err := helpers.AppEnvironment(opts.Provider, opts.App)
	if err != nil {
		return errors.WithStack(err)
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return errors.WithStack(err)
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
			return errors.WithStack(err)
		}

		o, err := opts.Provider.ObjectStore(opts.App, "", bytes.NewReader(data), structs.ObjectStoreOptions{})
		if err != nil {
			return errors.WithStack(err)
		}

		pw.Writef("build", "starting build\n")

		bopts := structs.BuildCreateOptions{Development: options.Bool(true)}

		if opts.Manifest != "" {
			bopts.Manifest = options.String(opts.Manifest)
		}

		b, err := opts.Provider.BuildCreate(opts.App, o.Url, bopts)
		if err != nil {
			return errors.WithStack(err)
		}

		logs, err := opts.Provider.BuildLogs(opts.App, b.Id, structs.LogsOptions{})
		if err != nil {
			return errors.WithStack(err)
		}

		bo := pw.Writer("build")

		go io.Copy(bo, logs)

		if err := opts.waitForBuild(ctx, b.Id); err != nil {
			return errors.WithStack(err)
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		b, err = opts.Provider.BuildGet(opts.App, b.Id)
		if err != nil {
			return errors.WithStack(err)
		}

		popts := structs.ReleasePromoteOptions{
			Development: options.Bool(true),
			Min:         options.Int(0),
		}

		if err := opts.Provider.ReleasePromote(opts.App, b.Release, popts); err != nil {
			return errors.WithStack(err)
		}
	}

	errch := make(chan error)
	defer close(errch)

	go handleErrors(ctx, pw, errch)

	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
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
		var buf bytes.Buffer

		if _, err := opts.Provider.ProcessExec(opts.App, pid, "pwd", &buf, structs.ProcessExecOptions{}); err != nil {
			return errors.WithStack(fmt.Errorf("%s pwd: %s", pid, err))
		}

		wd := strings.TrimSpace(buf.String())

		remote = filepath.Join(wd, remote)
	}

	rp, wp := io.Pipe()

	ch := make(chan error)

	go func() {
		ch <- opts.Provider.FilesUpload(opts.App, pid, rp)
		close(ch)
	}()

	tw := tar.NewWriter(wp)

	for _, add := range adds {
		local := filepath.Join(add.Base, add.Path)

		stat, err := os.Stat(local)
		if err != nil {
			// skip transient files like '.git/.COMMIT_EDITMSG.swp'
			if os.IsNotExist(err) {
				continue
			}

			return errors.WithStack(err)
		}

		tw.WriteHeader(&tar.Header{
			Name:    filepath.Join(remote, add.Path),
			Mode:    int64(stat.Mode()),
			Size:    stat.Size(),
			ModTime: stat.ModTime(),
		})

		fd, err := os.Open(local)
		if err != nil {
			return errors.WithStack(err)
		}

		defer fd.Close()

		if _, err := io.Copy(tw, fd); err != nil {
			return errors.WithStack(err)
		}

		fd.Close()
	}

	if err := tw.Close(); err != nil {
		return errors.WithStack(err)
	}

	if err := wp.Close(); err != nil {
		return errors.WithStack(err)
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
			defer res.Body.Close()

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
				return errors.WithStack(err)
			}

			switch b.Status {
			case "created", "running":
				break
			case "complete":
				return nil
			case "failed":
				return errors.WithStack(fmt.Errorf("build failed"))
			default:
				return errors.WithStack(fmt.Errorf("unknown build status: %s", b.Status))
			}
		}
	}
}

func (opts Options2) watchChanges(ctx context.Context, pw prefix.Writer, m *manifest.Manifest, service, root string, ch chan error) {
	bss, err := buildSources(m, root, service)
	if err != nil {
		ch <- fmt.Errorf("sync error: %s", err)
		return
	}

	ignores, err := buildIgnores(root, service)
	if err != nil {
		ch <- fmt.Errorf("sync error: %s", err)
		return
	}

	for _, bs := range bss {
		go opts.watchPath(ctx, pw, service, root, bs, ignores, ch)
	}
}

func (opts Options2) watchPath(ctx context.Context, pw prefix.Writer, service, root string, bs buildSource, ignores []string, ch chan error) {
	cch := make(chan changes.Change, 1)

	abs, err := filepath.Abs(bs.Local)
	if err != nil {
		ch <- fmt.Errorf("sync error: %s", err)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		ch <- fmt.Errorf("sync error: %s", err)
		return
	}

	rel, err := filepath.Rel(wd, bs.Local)
	if err != nil {
		ch <- fmt.Errorf("sync error: %s", err)
		return
	}

	pw.Writef("convox", "starting sync from <dir>%s</dir> to <dir>%s</dir> on <service>%s</service>\n", rel, helpers.CoalesceString(bs.Remote, "."), service)

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
					pw.Writef("convox", "sync: %d files to <dir>%s</dir> on <service>%s</service>\n", len(adds), helpers.CoalesceString(bs.Remote, "."), service)
				case len(adds) > 0:
					for _, a := range adds {
						pw.Writef("convox", "sync: <dir>%s</dir> to <dir>%s</dir> on <service>%s</service>\n", a.Path, helpers.CoalesceString(bs.Remote, "."), service)
					}
				}

				if err := opts.handleAdds(ps.Id, bs.Remote, adds); err != nil {
					pw.Writef("convox", "sync add error: %s\n", err)
				}

				switch {
				case len(removes) > 3:
					pw.Writef("convox", "remove: %d files from <dir>%s</dir> to <service>%s</service>\n", len(removes), helpers.CoalesceString(bs.Remote, "."), service)
				case len(removes) > 0:
					for _, r := range removes {
						pw.Writef("convox", "remove: <dir>%s</dir> from <dir>%s</dir> on <service>%s</service>\n", r.Path, helpers.CoalesceString(bs.Remote, "."), service)
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
		return nil, errors.WithStack(err)
	}

	if s.Image != "" {
		return nil, nil
	}

	path, err := filepath.Abs(filepath.Join(root, s.Build.Path, s.Build.Manifest))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.WithStack(fmt.Errorf("no such file: %s", filepath.Join(s.Build.Path, s.Build.Manifest)))
	}

	return ioutil.ReadFile(path)
}

func buildIgnores(root, service string) ([]string, error) {
	fd, err := os.Open(filepath.Join(root, ".dockerignore"))
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dockerignore.ReadAll(fd)
}

func buildSources(m *manifest.Manifest, root, service string) ([]buildSource, error) {
	data, err := buildDockerfile(m, root, service)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if data == nil {
		return []buildSource{}, nil
	}

	svc, err := m.Service(service)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	bs := []buildSource{}
	env := map[string]string{}
	wd := ""

	s := bufio.NewScanner(bytes.NewReader(data))

lines:
	for s.Scan() {
		parts := strings.Fields(s.Text())

		if len(parts) < 1 {
			continue
		}

		switch strings.ToUpper(parts[0]) {
		case "ADD", "COPY":
			for i, p := range parts {
				if m := reDockerOption.FindStringSubmatch(p); len(m) > 1 {
					switch strings.ToLower(m[1]) {
					case "from":
						continue lines
					default:
						parts = append(parts[:i], parts[i+1:]...)
					}
				}
			}

			if len(parts) > 2 {
				u, err := url.Parse(parts[1])
				if err != nil {
					return nil, errors.WithStack(err)
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
					continue
				}

				if err := json.Unmarshal(data, &ee); err != nil {
					return nil, errors.WithStack(err)
				}

				for _, e := range ee {
					parts := strings.SplitN(e, "=", 2)

					if len(parts) == 2 {
						env[parts[0]] = parts[1]
					}
				}

				data, err = Exec.Execute("docker", "inspect", parts[1], "--format", "{{.Config.WorkingDir}}")
				if err != nil {
					return nil, errors.WithStack(err)
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
			return nil, errors.WithStack(err)
		}

		stat, err := os.Stat(abs)
		if err != nil {
			return nil, errors.WithStack(err)
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
					return nil, errors.WithStack(err)
				}

				rr, err := filepath.Rel(bs[j].Remote, bs[i].Remote)
				if err != nil {
					return nil, errors.WithStack(err)
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

type stackTracer interface {
	StackTrace() errors.StackTrace
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

var ansiScreenSequences = []*regexp.Regexp{
	regexp.MustCompile("\033\\[\\d+;\\d+H"),
}

func stripANSIScreenCommands(data string) string {
	for _, r := range ansiScreenSequences {
		data = r.ReplaceAllString(data, "")
	}

	return data
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

			service := strings.Split(match[4], ":")[0]

			if !services[service] {
				continue
			}

			stripped := stripANSIScreenCommands(match[6])

			pw.Writef(service, "%s\n", stripped)
		}
	}

	if err := ls.Err(); err != nil {
		pw.Writef("convox", "scan error: %s\n", err)
	}
}
