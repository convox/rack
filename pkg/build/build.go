package build

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/exec"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest1"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
)

// Runtime constants allow us to avoid magic‑strings sprinkled everywhere
const (
	RuntimeDaemonless = "daemonless"
)

// Options holds the parameters for a single build execution
// (mirrors CLI/environment flags one‑for‑one).
// NOTE: keep struct tags aligned with main.go when you add new fields.
type Options struct {
	App         string
	Auth        string
	BuildArgs   []string
	Cache       bool
	Development bool
	EnvWrapper  bool
	Generation  string
	Id          string
	Manifest    string
	Output      io.Writer
	Push        string
	Rack        string
	Source      string
	Runtime     string
}

// Build represents a build session.  A value is short‑lived and NOT safe for reuse.
type Build struct {
	Options
	Exec     exec.Interface
	Provider structs.Provider

	logs   bytes.Buffer // collected build output
	writer io.Writer    // tee to Output + logs
}

// New prepares a Build instance but does NOT start any long‑running work.
func New(opts Options) (*Build, error) {
	b := &Build{Options: opts}

	// sane defaults ----------------------------------------------------------
	if b.Manifest == "" {
		if b.Generation == "2" {
			b.Manifest = "convox.yml"
		} else {
			b.Manifest = "docker-compose.yml"
		}
	}

	// dependencies -----------------------------------------------------------
	client, err := sdk.NewFromEnv()
	if err != nil {
		return nil, fmt.Errorf("initialising provider: %w", err)
	}
	b.Provider = client

	b.Exec = &exec.Exec{}

	// log fan‑out: always capture into buffer + optionally caller‑supplied writer
	if opts.Output != nil {
		b.writer = io.MultiWriter(opts.Output, &b.logs)
	} else {
		b.writer = io.MultiWriter(os.Stdout, &b.logs)
	}

	return b, nil
}

// Execute runs the entire build pipeline; it is the public entrypoint.
func (bb *Build) Execute() error {
	if err := bb.execute(); err != nil {
		return bb.fail(err) // fail() already logs + updates provider
	}
	return nil
}

// printf helper that always targets the build‐scoped writer
func (bb *Build) Printf(format string, args ...interface{}) { // nolint: revive  — this is intentional
	fmt.Fprintf(bb.writer, format, args...)
}

// ---------------------------------------------------------------------------
// private helpers below
// ---------------------------------------------------------------------------

func (bb *Build) execute() error {
	// 1. verify build exists --------------------------------------------------
	if _, err := bb.Provider.BuildGet(bb.App, bb.Id); err != nil {
		return fmt.Errorf("checking build record: %w", err)
	}

	// 2. optional docker login (classic runtime only) ------------------------
	if bb.Runtime != RuntimeDaemonless {
		if err := bb.login(); err != nil {
			return err // already wrapped inside login()
		}
	}

	// 3. workspace prep ------------------------------------------------------
	dir, err := os.MkdirTemp("", "convox-build-")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	// ensure cleanup *and* cwd restore even on panic
	defer os.RemoveAll(dir)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("detecting cwd: %w", err)
	}
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("chdir to workspace: %w", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	// 4. fetch & unarchive source -------------------------------------------
	u, err := url.Parse(bb.Source)
	if err != nil {
		return fmt.Errorf("parsing source url: %w", err)
	}
	if u.Scheme != "object" {
		return fmt.Errorf("only object:// sources are supported (got %s)", u.Scheme)
	}

	objReader, err := bb.Provider.ObjectFetch(u.Host, u.Path)
	if err != nil {
		return fmt.Errorf("fetching source object: %w", err)
	}
	gz, err := gzip.NewReader(objReader)
	if err != nil {
		return fmt.Errorf("opening source gzip: %w", err)
	}
	if err := helpers.Unarchive(gz, "."); err != nil {
		return fmt.Errorf("unarchive source: %w", err)
	}

	// 5. save manifest contents for audit ------------------------------------
	manifestBytes, err := os.ReadFile(bb.Manifest)
	if err != nil {
		return fmt.Errorf("reading manifest %s: %w", bb.Manifest, err)
	}
	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{
		Manifest: options.String(string(manifestBytes)),
	}); err != nil {
		return fmt.Errorf("persist manifest: %w", err)
	}

	// 6. route to generation/runtime implementation --------------------------
	switch {
	case bb.Generation == "2" && bb.Runtime == RuntimeDaemonless:
		if err := bb.buildGeneration2Daemonless("."); err != nil {
			return err
		}
	case bb.Generation == "2":
		if err := bb.buildGeneration2("."); err != nil {
			return err
		}
	default:
		if err := bb.buildGeneration1("."); err != nil {
			return err
		}
	}

	// 7. mark success --------------------------------------------------------
	return bb.success()
}

// login performs docker login for each registry entry in Auth JSON.
func (bb *Build) login() error {
	if strings.TrimSpace(bb.Auth) == "" { // nothing to do
		return nil
	}

	var auth map[string]struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(bb.Auth), &auth); err != nil {
		return fmt.Errorf("parsing docker auth json: %w", err)
	}

	for host, entry := range auth {
		buf := &bytes.Buffer{}
		if err := bb.Exec.Stream(buf, strings.NewReader(entry.Password), "docker", "login", "-u", entry.Username, "--password-stdin", host); err != nil {
			return fmt.Errorf("docker login %s: %w", host, err)
		}
		bb.Printf("Authenticating %s: %s\n", host, strings.TrimSpace(buf.String()))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Generation‑specific helpers (generation 1 delegated here for brevity)
// ---------------------------------------------------------------------------

func (bb *Build) buildGeneration1(dir string) error {
	dcy := filepath.Join(dir, bb.Manifest)

	if _, err := os.Stat(dcy); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", bb.Manifest)
	}

	data, err := os.ReadFile(dcy)
	if err != nil {
		return fmt.Errorf("read compose manifest: %w", err)
	}

	m, err := manifest1.Load(data)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	if verrs := m.Validate(); len(verrs) > 0 {
		return verrs[0]
	}

	// set up streaming of log lines from manifest build
	lines := make(chan string)
	go func() {
		for l := range lines {
			bb.Printf("%s\n", l)
		}
	}()
	defer close(lines)

	env, err := helpers.AppEnvironment(bb.Provider, bb.App)
	if err != nil {
		return fmt.Errorf("load app env: %w", err)
	}
	for _, v := range bb.BuildArgs {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid build arg: %s", v)
		}
		env[parts[0]] = parts[1]
	}

	if err := m.Build(dir, bb.App, lines, manifest1.BuildOptions{Environment: env, Cache: bb.Cache}); err != nil {
		return err // manifest already emits details to logs
	}

	return m.Push(bb.Push, bb.App, bb.Id, lines)
}

// ---------------------------------------------------------------------------
// success / failure helpers --------------------------------------------------
// ---------------------------------------------------------------------------

func (bb *Build) success() error {
	// upload logs ------------------------------------------------------------
	obj, err := bb.Provider.ObjectStore(bb.App, fmt.Sprintf("build/%s/logs", bb.Id), bytes.NewReader(bb.logs.Bytes()), structs.ObjectStoreOptions{})
	if err != nil {
		return fmt.Errorf("store logs: %w", err)
	}

	// close out build --------------------------------------------------------
	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{
		Ended: options.Time(time.Now().UTC()),
		Logs:  options.String(obj.Url),
	}); err != nil {
		return fmt.Errorf("final build update: %w", err)
	}

	// create release ---------------------------------------------------------
	rel, err := bb.Provider.ReleaseCreate(bb.App, structs.ReleaseCreateOptions{Build: options.String(bb.Id)})
	if err != nil {
		return fmt.Errorf("create release: %w", err)
	}

	if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{
		Release: options.String(rel.Id),
		Status:  options.String("complete"),
	}); err != nil {
		return fmt.Errorf("link release: %w", err)
	}

	bb.Provider.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": bb.App, "id": bb.Id, "release_id": rel.Id}})
	return nil
}

func (bb *Build) fail(buildErr error) error { // always returns buildErr
	bb.Printf("ERROR: %s\n", buildErr)

	bb.Provider.EventSend("build:create", structs.EventSendOptions{
		Data:  map[string]string{"app": bb.App, "id": bb.Id},
		Error: options.String(buildErr.Error()),
	})

	obj, err := bb.Provider.ObjectStore(bb.App, fmt.Sprintf("build/%s/logs", bb.Id), bytes.NewReader(bb.logs.Bytes()), structs.ObjectStoreOptions{})
	if err != nil {
		return fmt.Errorf("store failure logs: %w", err)
	}

	_, _ = bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{
		Ended:  options.Time(time.Now().UTC()),
		Logs:   options.String(obj.Url),
		Status: options.String("failed"),
	})

	return buildErr // propagate original error for caller handling
}
