package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/convox/rack/api/cmd/build/source"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/provider"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}

var (
	flagApp    string
	flagAuth   string
	flagId     string
	flagConfig string
	flagMethod string
	flagPush   string
	flagUrl    string

	currentBuild *structs.Build
)

func init() {
	flag.StringVar(&flagApp, "app", "example", "app name")
	flag.StringVar(&flagAuth, "auth", "", "docker auth data (base64 encoded)")
	flag.StringVar(&flagConfig, "config", "docker-compose.yml", "path to app config")
	flag.StringVar(&flagId, "id", "latest", "build id")
	flag.StringVar(&flagMethod, "method", "", "source method")
	flag.StringVar(&flagPush, "push", "", "push to registry")
	flag.StringVar(&flagUrl, "url", "", "source url")
}

func main() {
	flag.Parse()

	if v := os.Getenv("BUILD_APP"); v != "" {
		flagApp = v
	}

	if v := os.Getenv("BUILD_AUTH"); v != "" {
		flagAuth = v
	}

	if v := os.Getenv("BUILD_CONFIG"); v != "" {
		flagConfig = v
	}

	if v := os.Getenv("BUILD_ID"); v != "" {
		flagId = v
	}

	if v := os.Getenv("BUILD_PUSH"); v != "" {
		flagPush = v
	}

	if v := os.Getenv("BUILD_URL"); v != "" {
		flagUrl = v
	}

	// fmt.Printf("flagApp = %+v\n", flagApp)
	// fmt.Printf("flagAuth = %+v\n", flagAuth)
	// fmt.Printf("flagConfig = %+v\n", flagConfig)
	// fmt.Printf("flagId = %+v\n", flagId)
	// fmt.Printf("flagMethod = %+v\n", flagMethod)
	// fmt.Printf("flagPush = %+v\n", flagPush)
	// fmt.Printf("flagUrl = %+v\n", flagUrl)

	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)

		currentBuild.Status = "failed"

		if err := provider.FromEnv().BuildSave(currentBuild); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		}

		os.Exit(1)
	}

	currentBuild.Status = "complete"

	if err := provider.FromEnv().BuildSave(currentBuild); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func execute() error {
	b, err := provider.FromEnv().BuildGet(flagApp, flagId)
	if err != nil {
		return err
	}

	currentBuild = b

	dir, err := fetch()
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	if err := login(); err != nil {
		return err
	}

	if err := build(dir); err != nil {
		return err
	}

	return nil
}

func fetch() (string, error) {
	var s source.Source

	switch flagMethod {
	case "tgz":
		s = &source.SourceTgz{flagUrl}
	default:
		die(fmt.Errorf("unknown method: %s", flagMethod))
	}

	return s.Fetch()
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
		fmt.Printf("Authenticating %s: ", host)

		cmd := exec.Command("docker", "login", "-u", entry.Username, "-p", entry.Password, host)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func build(dir string) error {
	dcy := filepath.Join(dir, "docker-compose.yml")

	if _, err := os.Stat(dcy); os.IsNotExist(err) {
		return fmt.Errorf("no docker-compose.yml found")
	}

	data, err := ioutil.ReadFile(dcy)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data)
	if err != nil {
		return err
	}

	s := make(chan string)

	go func() {
		for l := range s {
			fmt.Println(l)
		}
	}()

	if err := m.Build(dir, flagApp, s, true); err != nil {
		return err
	}

	if err := m.Push(flagPush, flagApp, flagId, s); err != nil {
		return err
	}

	return nil
}
