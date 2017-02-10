package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/cmd/convox/templates"
	"github.com/convox/rack/manifest"
	"gopkg.in/urfave/cli.v1"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "init",
		Description: "initialize an app for local development",
		Usage:       "[directory]",
		Action:      cmdInit,
	})
}

// EnvEntry is an environment entry from an app.json file
type EnvEntry struct {
	Value string
}

// AppManifest represent specific fields of an app.json file
type AppManifest struct {
	Addons []string
	Env    map[string]EnvEntry
}

var appFound bool // flag if an actual app.json file is present

// ProcfileEntry is an entry in a Procfile
type ProcfileEntry struct {
	Name    string
	Command string
}

// Procfile represents a Procfile used in Heroku-based apps
type Procfile []ProcfileEntry

var reProcfile = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

// Release is type representing output buildback release script
type Release struct {
	Addons       []string
	ProcessTypes map[string]string `yaml:"default_process_types"`
}

func cmdInit(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctID, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err})
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	if helpers.Exists(path.Join(dir, "docker-compose.yml")) {
		fmt.Println("found docker-compose.yml, try `convox start` instead")
		return nil
	}

	appType, err := initApplication(dir)
	if err != nil {
		stdcli.QOSEventSend("Dev Code Update Failed", distinctID, stdcli.QOSEventProperties{Error: err, AppType: appType})
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err, AppType: appType})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("Dev Code Updated", distinctID, stdcli.QOSEventProperties{AppType: appType})
	stdcli.QOSEventSend("cli-init", distinctID, ep)
	return nil
}

// appKind maps to the various buildpacks and their detect output
var appKind = map[string]string{
	"Node.js": "nodejs",
	"Python":  "python",
	"Ruby":    "ruby",
}

func initApplication(dir string) (string, error) {
	prepURL := "https://convox.com/docs/preparing-an-application/"
	args := []string{"run", "--rm", "--name", "convox-init", "-v", fmt.Sprintf("%s:/tmp/app", dir), "convox/init"}

	k, err := exec.Command(dockerBin, append(args, "detect")...).Output()
	if err != nil {
		return "", fmt.Errorf("unable to detect app type: convox/init - %s", err)
	}

	kd := strings.TrimSpace(string(k))
	kind, ok := appKind[kd]
	if !ok {
		return kind, fmt.Errorf("unknown app type: %s \ncheck out %s for more information", kd, prepURL)
	}

	fmt.Printf("initializing a %s app\n", kind)

	if err := writeAsset("entrypoint.sh", "buildpack/entrypoint.sh", nil); err != nil {
		return kind, err
	}

	input := map[string]interface{}{
		"framework":   kind,
		"environment": buildpackEnvironment(kind),
	}
	if err := writeAsset("Dockerfile", "buildpack/Dockerfile", input); err != nil {
		return kind, err
	}

	if err := writeAsset(".dockerignore", "buildpack/.dockerignore", nil); err != nil {
		return kind, err
	}

	// docker-compose.yml
	data, err := generateManifestData(dir, kind)
	if err != nil {
		return kind, err
	}

	if err := writeFile("docker-compose.yml", data, 0644); err != nil {
		return kind, err
	}

	fmt.Println()
	fmt.Println("try running `convox start`")
	return kind, err
}

// ReadAppManifest reads data that follows the app.json manifest format
func ReadAppManifest(data []byte) (AppManifest, error) {
	am := AppManifest{
		[]string{},
		nil,
	}

	if err := json.Unmarshal(data, &am); err != nil {
		return am, err
	}

	return am, nil
}

// readAppfile reads a file and returns an AppManifest
func readAppfile(path string) (AppManifest, error) {
	var am AppManifest

	if helpers.Exists(path) {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return am, err
		}

		am, err = ReadAppManifest(data)
		if err != nil {
			return am, err
		}
		appFound = true
	}

	return am, nil
}

// ReadProcfile reads data that follows the Procfile format
func ReadProcfile(data []byte) Procfile {
	pf := Procfile{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := reProcfile.FindStringSubmatch(scanner.Text())

		if len(parts) == 3 {
			pf = append(pf, ProcfileEntry{
				Name:    parts[1],
				Command: parts[2],
			})
		}
	}
	return pf
}

// readProcfile reads a file and returns an Procfile
func readProcfile(path string) (Procfile, error) {
	if !helpers.Exists(path) {
		return Procfile{}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ReadProcfile(data), nil
}

// buildpackEnvironment creates environment variables that are required to run a buildpack
func buildpackEnvironment(kind string) map[string]string {
	switch kind {
	case "ruby":
		return map[string]string{
			"CURL_CONNECT_TIMEOUT": "0", // default timeouts for curl are too aggressive causing failure
			"CURL_TIMEOUT":         "0",
			"STACK":                "cedar-14",
		}
	default:
		return map[string]string{}
	}
}

// GenerateManifest generates a Manifest from the union of a Procfile, AppManifest and Release data
func GenerateManifest(pf Procfile, am AppManifest, r Release) manifest.Manifest {

	m := manifest.Manifest{
		Services: make(map[string]manifest.Service),
		Version:  "2",
	}

	if len(pf) == 0 {
		for name, cmd := range r.ProcessTypes {
			me := manifest.Service{
				Build: manifest.Build{
					Context: ".",
				},
				Command: manifest.Command{
					String: cmd,
				},
			}

			m.Services[name] = me
		}
	}

	for _, e := range pf {
		me := manifest.Service{
			Build: manifest.Build{
				Context: ".",
			},
			Command: manifest.Command{
				String: e.Command,
			},
			Environment: manifest.Environment{},
			Labels:      make(manifest.Labels),
			Ports:       make(manifest.Ports, 0),
		}

		switch e.Name {
		case "web":
			me.Name = "web"
			me.Labels["convox.port.443.protocol"] = "tls"

			me.Ports = append(me.Ports, manifest.Port{
				Name:      "80",
				Balancer:  80,
				Container: 4001,
				Public:    true,
				Protocol:  manifest.TCP,
			})
			me.Ports = append(me.Ports, manifest.Port{
				Name:      "443",
				Balancer:  443,
				Container: 4001,
				Public:    true,
				Protocol:  manifest.TCP,
			})

			me.Environment = append(me.Environment, manifest.EnvironmentItem{
				Name:  "PORT",
				Value: "4001",
			})

			for k, v := range am.Env {
				me.Environment = append(me.Environment, manifest.EnvironmentItem{
					Name:  k,
					Value: v.Value,
				})
			}
		}
		m.Services[e.Name] = me
	}
	return m
}

// generateManifestData creates a Manifest from files in the directory
func generateManifestData(dir, kind string) ([]byte, error) {
	pf, err := readProcfile(path.Join(dir, "Procfile"))
	if err != nil {
		return nil, err
	}

	am, err := readAppfile(path.Join(dir, "app.json"))
	if err != nil {
		return nil, err
	}

	var release Release
	if len(pf) == 0 || !appFound {
		var r []byte
		args := []string{"run", "--rm", "--name", "convox-init", "-v", fmt.Sprintf("%s:/tmp/app", dir), "convox/init"}

		// NOTE: The ruby-buildpack generates a yaml file during compile so we have to perform both steps
		if kind == "ruby" {
			// this can be time consuming, let's give feedback
			stdcli.Spinner.Prefix = "generating ruby data: "
			stdcli.Spinner.Start()
			r, err = exec.Command(dockerBin, append(args, "compile-release")...).Output()
			fmt.Printf("\x08\x08OK\n")
			stdcli.Spinner.Stop()
		} else {
			r, err = exec.Command(dockerBin, append(args, "release")...).Output()
		}
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(r, &release); err != nil {
			return nil, err
		}
	}

	m := GenerateManifest(pf, am, release)

	adds := []string{}
	if appFound {
		adds = append(adds, am.Addons...)
	} else {
		adds = append(adds, release.Addons...)
	}
	ParseAddons(adds, &m)

	return yaml.Marshal(m)
}

// writeFile is a helper function that writes a file
func writeFile(path string, data []byte, mode os.FileMode) error {
	fmt.Printf("writing %s... ", path)

	if helpers.Exists(path) {
		fmt.Println("EXISTS")
		return nil
	}

	// make the containing directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, data, mode); err != nil {
		return err
	}

	fmt.Println("OK")

	return nil
}

// writeAsset is a helper function that generates an asset and writes a file
func writeAsset(path, templateName string, input map[string]interface{}) error {
	data, err := templates.Asset(templateName)
	if err != nil {
		return err
	}

	info, err := templates.AssetInfo(templateName)
	if err != nil {
		return err
	}

	if input != nil {
		tmpl, err := template.New(templateName).Parse(string(data))
		if err != nil {
			return err
		}

		var formation bytes.Buffer

		err = tmpl.Execute(&formation, input)
		if err != nil {
			return err
		}

		data = formation.Bytes()
	}

	return writeFile(path, data, info.Mode())
}

// AddonHandler is a func type to handle addons
type AddonHandler func(m *manifest.Manifest)

// ParseAddons iterates through an apps addons and edits the manifest accordingly
func ParseAddons(addons []string, m *manifest.Manifest) {
	handlers := map[string]AddonHandler{
		"heroku-postgresql": postgresAddon,
	}

	for _, name := range addons {
		if f, ok := handlers[name]; ok {
			f(m)
		}
		// TODO: event on unknown addons?
	}
}

func postgresAddon(m *manifest.Manifest) {
	s := manifest.Service{
		Image: "convox/postgres",
		Name:  "database",
		Ports: manifest.Ports{
			{
				Balancer:  5432,
				Container: 5432,
				Public:    false,
				Protocol:  manifest.TCP,
			},
		},
		Volumes: []string{
			"/var/lib/postgresql/data",
		},
	}

	web := m.Services["web"]
	web.Links = append(web.Links, "database")
	m.Services["web"] = web

	m.Services["database"] = s
}
