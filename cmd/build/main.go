package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/rack/cmd/build/source"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/manifest1"
	"github.com/convox/rack/options"
	"github.com/convox/rack/sdk"
	"github.com/convox/rack/structs"
)

var (
	flagApp        string
	flagAuth       string
	flagCache      string
	flagGeneration string
	flagID         string
	flagManifest   string
	flagMethod     string
	flagPush       string
	flagRack       string
	flagUrl        string

	currentBuild    *structs.Build
	currentLogs     string
	currentManifest string

	rack *sdk.Client
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.StringVar(&flagApp, "app", "example", "app name")
	fs.StringVar(&flagAuth, "auth", "", "docker auth data (json)")
	fs.StringVar(&flagCache, "cache", "true", "use docker cache")
	fs.StringVar(&flagGeneration, "generation", "", "app generation")
	fs.StringVar(&flagID, "id", "latest", "build id")
	fs.StringVar(&flagManifest, "manifest", "", "path to app manifest")
	fs.StringVar(&flagMethod, "method", "", "source method")
	fs.StringVar(&flagPush, "push", "", "push to registry")
	fs.StringVar(&flagRack, "rack", "convox", "rack name")
	fs.StringVar(&flagUrl, "url", "", "source url")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fail(err)
	}

	if v := os.Getenv("BUILD_APP"); v != "" {
		flagApp = v
	}

	if v := os.Getenv("BUILD_AUTH"); v != "" {
		flagAuth = v
	}

	if v := os.Getenv("BUILD_GENERATION"); v != "" {
		flagGeneration = v
	}

	if v := os.Getenv("BUILD_ID"); v != "" {
		flagID = v
	}

	if v := os.Getenv("BUILD_MANIFEST"); v != "" {
		flagManifest = v
	}

	if v := os.Getenv("BUILD_PUSH"); v != "" {
		flagPush = v
	}

	if v := os.Getenv("BUILD_RACK"); v != "" {
		flagRack = v
	}

	if v := os.Getenv("BUILD_URL"); v != "" {
		flagUrl = v
	}

	if flagManifest == "" {
		switch flagGeneration {
		case "2":
			flagManifest = "convox.yml"
		default:
			flagManifest = "docker-compose.yml"
		}
	}

	var err error

	rack, err = sdk.NewFromEnv()
	if err != nil {
		fail(err)
	}

	if err := execute(); err != nil {
		fail(err)
	}

	if err := success(); err != nil {
		fail(err)
	}

	rack.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": flagApp, "id": flagID, "release_id": currentBuild.Release}})

	clean()

	time.Sleep(1 * time.Second)
}

func execute() error {
	b, err := rack.BuildGet(flagApp, flagID)
	if err != nil {
		return err
	}

	currentBuild = b

	if err := login(); err != nil {
		return err
	}

	dir, err := fetch()
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	data, err := ioutil.ReadFile(filepath.Join(dir, flagManifest))
	if err != nil {
		return err
	}

	currentBuild.Manifest = string(data)

	switch flagGeneration {
	case "2":
		if err := build2(dir); err != nil {
			return err
		}
	default:
		if err := build(dir); err != nil {
			return err
		}
	}

	return nil
}

func fetch() (string, error) {
	var s source.Source

	switch flagMethod {
	case "git":
		s = &source.SourceGit{flagUrl}
	case "tgz":
		s = &source.SourceTgz{flagUrl}
	case "zip":
		s = &source.SourceZip{flagUrl}
	default:
		return "", fmt.Errorf("unknown method: %s", flagMethod)
	}

	var buf bytes.Buffer

	dir, err := s.Fetch(&buf)
	logf(buf.String())
	if err != nil {
		return "", err
	}

	return dir, nil
}

func login() error {
	var auth map[string]struct {
		Username string
		Password string
	}

	if err := json.Unmarshal([]byte(flagAuth), &auth); err != nil {
		return err
	}

	for host, entry := range auth {
		out, err := exec.Command("docker", "login", "-u", entry.Username, "-p", entry.Password, host).CombinedOutput()
		log(fmt.Sprintf("Authenticating %s: %s", host, strings.TrimSpace(string(out))))
		if err != nil {
			return err
		}
	}

	return nil
}

func build(dir string) error {
	dcy := filepath.Join(dir, flagManifest)

	if _, err := os.Stat(dcy); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", flagManifest)
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
			log(l)
		}
	}()

	defer close(s)

	env, err := helpers.AppEnvironment(rack, flagApp)
	if err != nil {
		return err
	}

	err = m.Build(dir, flagApp, s, manifest1.BuildOptions{
		Environment: env,
		Cache:       flagCache == "true",
		Verbose:     false,
	})
	if err != nil {
		return err
	}

	if err := m.Push(flagPush, flagApp, flagID, s); err != nil {
		return err
	}

	return nil
}

func build2(dir string) error {
	config := filepath.Join(dir, flagManifest)

	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", flagManifest)
	}

	data, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	env, err := helpers.AppEnvironment(rack, flagApp)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return err
	}

	r, w := io.Pipe()

	go func() {
		buf := make([]byte, 1024)

		for {
			n, err := r.Read(buf)
			if err != nil {
				if err != io.EOF {
					log(fmt.Sprintf("ERROR: %s\n", err))
				}
				return
			}
			line := string(buf[0:n])
			currentLogs += line
			fmt.Print(line)
		}
	}()

	prefix := fmt.Sprintf("%s/%s", flagRack, flagApp)

	err = m.Build(prefix, flagID, manifest.BuildOptions{
		Cache:  flagCache == "true",
		Env:    env,
		Push:   flagPush,
		Root:   dir,
		Stdout: w,
		Stderr: w,
	})
	if err != nil {
		return err
	}

	return nil
}

func success() error {
	logs, err := rack.ObjectStore(flagApp, fmt.Sprintf("build/%s/logs", currentBuild.Id), bytes.NewReader([]byte(currentLogs)), structs.ObjectStoreOptions{})
	if err != nil {
		return err
	}

	opts := structs.BuildUpdateOptions{
		Ended:    options.Time(time.Now()),
		Logs:     options.String(logs.Url),
		Manifest: options.String(currentBuild.Manifest),
	}

	if _, err := rack.BuildUpdate(flagApp, currentBuild.Id, opts); err != nil {
		return err
	}

	r, err := rack.ReleaseCreate(flagApp, structs.ReleaseCreateOptions{Build: options.String(currentBuild.Id)})
	if err != nil {
		return err
	}

	opts = structs.BuildUpdateOptions{
		Release: options.String(r.Id),
		Status:  options.String("complete"),
	}

	if _, err := rack.BuildUpdate(flagApp, currentBuild.Id, opts); err != nil {
		return err
	}

	return nil
}

func fail(err error) {
	log(fmt.Sprintf("ERROR: %s", err))

	rack.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": flagApp, "id": flagID, "release_id": currentBuild.Release}, Error: err.Error()})

	logs, _ := rack.ObjectStore(flagApp, fmt.Sprintf("build/%s/logs", currentBuild.Id), bytes.NewReader([]byte(currentLogs)), structs.ObjectStoreOptions{})

	now := time.Now()
	status := "failed"

	opts := structs.BuildUpdateOptions{
		Ended:  options.Time(now),
		Logs:   options.String(logs.Url),
		Status: options.String(status),
	}

	if _, err := rack.BuildUpdate(flagApp, currentBuild.Id, opts); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}

	os.Exit(1)
}

func log(line string) {
	logf("%s\n", line)
}

func logf(f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	currentLogs += s
	fmt.Print(s)
}
