package build

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	// "os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/exec"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/manifest1"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
)

type Options struct {
	App         string
	Auth        string
	Cache       bool
	Development bool
	Generation  string
	Id          string
	Manifest    string
	Output      io.Writer
	Push        string
	Rack        string
	Source      string
}

type Build struct {
	Options
	Exec     exec.Interface
	Provider structs.Provider
	logs     bytes.Buffer
	writer   io.Writer
}

func New(opts Options) (*Build, error) {
	b := &Build{Options: opts}

	b.Exec = &exec.Exec{}

	if b.Manifest == "" {
		switch b.Generation {
		case "2":
			b.Manifest = "convox.yml"
		default:
			b.Manifest = "docker-compose.yml"
		}
	}

	r, err := sdk.NewFromEnv()
	if err != nil {
		return nil, err
	}

	b.Provider = r

	b.logs = bytes.Buffer{}

	if opts.Output != nil {
		b.writer = io.MultiWriter(opts.Output, &b.logs)
	} else {
		b.writer = io.MultiWriter(os.Stdout, &b.logs)
	}

	return b, nil
}

func (bb *Build) Execute() error {
	if err := bb.execute(); err != nil {
		return bb.fail(err)
	}

	return nil
}

func (bb *Build) Printf(format string, args ...interface{}) {
	fmt.Fprintf(bb.writer, format, args...)
}

func (bb *Build) execute() error {
	if _, err := bb.Provider.BuildGet(bb.App, bb.Id); err != nil {
		return err
	}

	if err := bb.login(); err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(dir); err != nil {
		return err
	}
	defer os.Chdir(cwd)

	u, err := url.Parse(bb.Source)
	if err != nil {
		return err
	}

	if u.Scheme != "object" {
		return fmt.Errorf("only object:// sources are supported")
	}

	r, err := bb.Provider.ObjectFetch(u.Host, u.Path)
	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	if err := helpers.Unarchive(gz, "."); err != nil {
		return err
	}

	data, err := ioutil.ReadFile(bb.Manifest)
	if err != nil {
		return err
	}

	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{Manifest: options.String(string(data))}); err != nil {
		return err
	}

	switch bb.Generation {
	case "2":
		if err := bb.buildGeneration2("."); err != nil {
			return err
		}
	default:
		if err := bb.buildGeneration1("."); err != nil {
			return err
		}
	}

	if err := bb.success(); err != nil {
		return err
	}

	return nil
}

func (bb *Build) login() error {
	var auth map[string]struct {
		Username string
		Password string
	}

	if err := json.Unmarshal([]byte(bb.Auth), &auth); err != nil {
		return err
	}

	for host, entry := range auth {
		buf := &bytes.Buffer{}

		err := bb.Exec.Stream(buf, strings.NewReader(entry.Password), "docker", "login", "-u", entry.Username, "--password-stdin", host)

		bb.Printf("Authenticating %s: %s\n", host, strings.TrimSpace(buf.String()))

		if err != nil {
			return err
		}
	}

	return nil
}

func (bb *Build) buildGeneration1(dir string) error {
	dcy := filepath.Join(dir, bb.Manifest)

	if _, err := os.Stat(dcy); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", bb.Manifest)
	}

	data, err := ioutil.ReadFile(dcy)
	if err != nil {
		return err
	}

	m, err := manifest1.Load(data)
	if err != nil {
		return err
	}

	errs := m.Validate()
	if len(errs) > 0 {
		return errs[0]
	}

	s := make(chan string)

	go func() {
		for l := range s {
			bb.Printf("%s\n", l)
		}
	}()

	defer close(s)

	env, err := helpers.AppEnvironment(bb.Provider, bb.App)
	if err != nil {
		return err
	}

	err = m.Build(dir, bb.App, s, manifest1.BuildOptions{
		Environment: env,
		Cache:       bb.Cache,
		Verbose:     false,
	})
	if err != nil {
		return err
	}

	if err := m.Push(bb.Push, bb.App, bb.Id, s); err != nil {
		return err
	}

	return nil
}

func (bb *Build) buildGeneration2(dir string) error {
	config := filepath.Join(dir, bb.Manifest)

	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", bb.Manifest)
	}

	data, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	env, err := helpers.AppEnvironment(bb.Provider, bb.App)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("%s/%s", bb.Rack, bb.App)

	builds := map[string]manifest.ServiceBuild{}
	pulls := map[string]bool{}
	pushes := map[string]string{}
	tags := map[string][]string{}

	for _, s := range m.Services {
		hash := s.BuildHash(bb.Id)
		to := fmt.Sprintf("%s/%s:%s", prefix, s.Name, bb.Id)

		if s.Image != "" {
			pulls[s.Image] = true
			tags[s.Image] = append(tags[s.Image], to)
		} else {
			builds[hash] = s.Build
			tags[hash] = append(tags[hash], to)
		}

		if bb.Push != "" {
			pushes[to] = fmt.Sprintf("%s:%s.%s", bb.Push, s.Name, bb.Id)
		}
	}

	for hash, b := range builds {
		bb.Printf("Building: %s\n", b.Path)

		if err := bb.build(filepath.Join(dir, b.Path), b.Manifest, hash, env); err != nil {
			return err
		}
	}

	for image := range pulls {
		bb.Printf("Pulling: %s\n", image)

		if err := bb.pull(image); err != nil {
			return err
		}
	}

	tagfroms := []string{}

	for from := range tags {
		tagfroms = append(tagfroms, from)
	}

	sort.Strings(tagfroms)

	for _, from := range tagfroms {
		tos := tags[from]

		for _, to := range tos {
			if err := bb.tag(from, to); err != nil {
				return err
			}

			if !bb.Development {
				if err := bb.injectConvoxEnv(to); err != nil {
					return err
				}
			}
		}
	}

	pushfroms := []string{}

	for from := range pushes {
		pushfroms = append(pushfroms, from)
	}

	sort.Strings(pushfroms)

	for _, from := range pushfroms {
		to := pushes[from]

		if err := bb.tag(from, to); err != nil {
			return err
		}

		bb.Printf("Pushing: %s\n", to)

		if err := bb.push(to); err != nil {
			return err
		}
	}

	return nil
}

func (bb *Build) success() error {
	logs, err := bb.Provider.ObjectStore(bb.App, fmt.Sprintf("build/%s/logs", bb.Id), bytes.NewReader(bb.logs.Bytes()), structs.ObjectStoreOptions{})
	if err != nil {
		return err
	}

	opts := structs.BuildUpdateOptions{
		Ended: options.Time(time.Now().UTC()),
		Logs:  options.String(logs.Url),
	}

	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, opts); err != nil {
		return err
	}

	r, err := bb.Provider.ReleaseCreate(bb.App, structs.ReleaseCreateOptions{Build: options.String(bb.Id)})
	if err != nil {
		return err
	}

	opts = structs.BuildUpdateOptions{
		Release: options.String(r.Id),
		Status:  options.String("complete"),
	}

	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, opts); err != nil {
		return err
	}

	bb.Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": bb.App, "id": bb.Id, "release_id": r.Id}})

	return nil
}

func (bb *Build) fail(buildError error) error {
	bb.Printf("ERROR: %s\n", buildError)

	bb.Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": bb.App, "id": bb.Id}, Error: options.String(buildError.Error())})

	logs, err := bb.Provider.ObjectStore(bb.App, fmt.Sprintf("build/%s/logs", bb.Id), bytes.NewReader(bb.logs.Bytes()), structs.ObjectStoreOptions{})
	if err != nil {
		return err
	}

	opts := structs.BuildUpdateOptions{
		Ended:  options.Time(time.Now().UTC()),
		Logs:   options.String(logs.Url),
		Status: options.String("failed"),
	}

	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, opts); err != nil {
		return err
	}

	return buildError
}
