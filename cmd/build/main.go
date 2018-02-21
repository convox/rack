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
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/options"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
)

var (
	flagApp        string
	flagAuth       string
	flagCache      string
	flagConfig     string
	flagGeneration string
	flagID         string
	flagMethod     string
	flagPush       string
	flagUrl        string

	currentBuild    *structs.Build
	currentLogs     string
	currentManifest string
	currentProvider structs.Provider

)

func init() {
	currentProvider = provider.FromEnv()

	var buf bytes.Buffer

	currentProvider.Initialize(structs.ProviderOptions{Logs: &buf})
}

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.StringVar(&flagApp, "app", "example", "app name")
	fs.StringVar(&flagAuth, "auth", "", "docker auth data (json)")
	fs.StringVar(&flagCache, "cache", "true", "use docker cache")
	fs.StringVar(&flagConfig, "config", "", "path to app config")
	fs.StringVar(&flagGeneration, "generation", "", "app generation")
	fs.StringVar(&flagID, "id", "latest", "build id")
	fs.StringVar(&flagMethod, "method", "", "source method")
	fs.StringVar(&flagPush, "push", "", "push to registry")
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

	if v := os.Getenv("BUILD_CONFIG"); v != "" {
		flagConfig = v
	}

	if v := os.Getenv("BUILD_GENERATION"); v != "" {
		flagGeneration = v
	}

	if v := os.Getenv("BUILD_ID"); v != "" {
		flagID = v
	}

	if v := os.Getenv("BUILD_PUSH"); v != "" {
		flagPush = v
	}

	if v := os.Getenv("BUILD_URL"); v != "" {
		flagUrl = v
	}

	if flagConfig == "" {
		switch flagGeneration {
		case "2":
			flagConfig = "convox.yml"
		default:
			flagConfig = "docker-compose.yml"
		}
	}

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
	b, err := currentProvider.BuildGet(flagApp, flagID)
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

	data, err := ioutil.ReadFile(filepath.Join(dir, flagConfig))
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
	log(strings.TrimSpace(buf.String()))
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
	dcy := filepath.Join(dir, flagConfig)

	if _, err := os.Stat(dcy); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", flagConfig)
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

	env, err := helpers.AppEnvironment(currentProvider, flagApp)
	if err != nil {
		return err
	}

	a, err := currentProvider.AppGet(flagApp)
	if err != nil {
		return err
	}

	sys, err := currentProvider.SystemGet()
	if err != nil {
		return err
	}

	env["SECURE_ENVIRONMENT_URL"] = a.Outputs["Environment"]
	env["SECURE_ENVIRONMENT_TYPE"] = "envfile"
	env["SECURE_ENVIRONMENT_KEY"] = sys.Outputs["EncryptionKey"]

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
	config := filepath.Join(dir, flagConfig)

	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", flagConfig)
	}

	data, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	env, err := helpers.AppEnvironment(currentProvider, flagApp)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data, manifest.Environment(env))
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

	err = m.Build(flagApp, flagID, manifest.BuildOptions{
		Env:    manifest.Environment(env),
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
	logs, err := currentProvider.ObjectStore(flagApp, fmt.Sprintf("build/%s/logs", currentBuild.Id), bytes.NewReader([]byte(currentLogs)), structs.ObjectStoreOptions{})
	if err != nil {
		return err
	}

	now := time.Now()
	status := "complete"

	opts := structs.BuildUpdateOptions{
		Ended:    options.Time(now),
		Logs:     options.String(logs.Url),
		Manifest: options.String(currentBuild.Manifest),
		Status:   options.String(status),
	}

	if _, err := currentProvider.BuildUpdate(flagApp, currentBuild.Id, opts); err != nil {
		return err
	}

	r, err := currentProvider.ReleaseCreate(flagApp, structs.ReleaseCreateOptions{Build: options.String(currentBuild.Id)})
	if err != nil {
		return err
	}

	if _, err := currentProvider.BuildUpdate(flagApp, currentBuild.Id, structs.BuildUpdateOptions{Release: options.String(r.Id)}); err != nil {
		return err
	}

	return nil
}

func fail(err error) {
	log(fmt.Sprintf("ERROR: %s", err))

	logs, _ := currentProvider.ObjectStore(flagApp, fmt.Sprintf("build/%s/logs", currentBuild.Id), bytes.NewReader([]byte(currentLogs)), structs.ObjectStoreOptions{})
	rack.EventSend("build:create", structs.EventSendOptions{Data: map[string]string{"app": flagApp, "id": flagID, "release_id": currentBuild.Release}, Error: err.Error()})


	now := time.Now()
	status := "failed"

	opts := structs.BuildUpdateOptions{
		Ended:  options.Time(now),
		Logs:   options.String(logs.Url),
		Status: options.String(status),
	}

	if _, err := currentProvider.BuildUpdate(flagApp, currentBuild.Id, opts); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}

	os.Exit(1)
}

func log(line string) {
	currentLogs += fmt.Sprintf("%s\n", line)
	fmt.Println(line)
}
