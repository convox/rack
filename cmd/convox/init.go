package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

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

func cmdInit(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	// TODO parse the Dockerfile and build a docker-compose.yml
	if exists("docker-compose.yml") {
		return stdcli.Error(fmt.Errorf("Cannot initialize a project that already contains docker-compose.yml"))

	}

	err = initApplication(dir)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctId, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("cli-init", distinctId, ep)
	return nil
}

func detectApplication(dir string) string {
	switch {
	case exists(filepath.Join(dir, "Procfile")):
		return "heroku"
	case exists(filepath.Join(dir, "manage.py")):
		return "django"
	case exists(filepath.Join(dir, "config/application.rb")):
		return "rails"
	case exists(filepath.Join(dir, "config.ru")):
		return "sinatra"
	case exists(filepath.Join(dir, "Gemfile.lock")):
		return "ruby"
	case exists(filepath.Join(dir, "requirements.txt")):
		return "python"
	}

	return "unknown"
}

func detectBuildpack(dir string) string {
	switch {
	case exists(filepath.Join(dir, "requirements.txt")) || exists(filepath.Join(dir, "setup.py")):
		return "python"
	case exists(filepath.Join(dir, "package.json")):
		return "nodejs"
	case exists(filepath.Join(dir, "Gemfile")):
		return "ruby"
	}

	return "unknown"
}

// buildpackEnvironment creates environment variables that are buildpack specific
func buildpackEnvironment(kind string) map[string]string {
	switch kind {
	case "ruby":
		return map[string]string{
			"CURL_CONNECT_TIMEOUT": "0", // default timeout is too aggressive causing failure
			"STACK":                "cedar-14",
		}
	default:
		return map[string]string{}
	}
}

func initApplication(dir string) error {
	// TODO parse the Dockerfile and build a docker-compose.yml
	if exists("Dockerfile") || exists("docker-compose.yml") {
		return nil
	}

	kind := detectApplication(dir)

	fmt.Printf("Initializing %s\n", kind)

	var input map[string]interface{}

	if kind == "heroku" {
		bp := detectBuildpack(dir)
		if bp == "unknown" {
			// TODO: track this event
			return fmt.Errorf("unable to detect Buildpack")
		}

		if err := writeAsset("entrypoint.sh", "init/buildpack/entrypoint.sh", nil); err != nil {
			return err
		}

		kind = "buildpack"

		input = map[string]interface{}{
			"framework":   bp,
			"environment": buildpackEnvironment(bp),
		}
	}

	if err := writeAsset("Dockerfile", fmt.Sprintf("init/%s/Dockerfile", kind), input); err != nil {
		return err
	}

	if err := generateManifest(dir, fmt.Sprintf("init/%s/docker-compose.yml", kind)); err != nil {
		return err
	}

	if err := writeAsset(".dockerignore", fmt.Sprintf("init/%s/.dockerignore", kind), input); err != nil {
		return err
	}

	return nil
}

func generateManifest(dir string, def string) error {
	if exists("Procfile") {
		pf, err := readProcfile("Procfile")
		if err != nil {
			return err
		}

		m := manifest.Manifest{
			Services: make(map[string]manifest.Service),
		}

		for _, e := range pf {
			me := manifest.Service{
				Build: manifest.Build{
					Context: ".",
				},
				Command: manifest.Command{
					String: e.Command,
				},
				Environment: make(manifest.Environment),
				Labels:      make(manifest.Labels),
				Ports:       make(manifest.Ports, 0),
			}

			switch e.Name {
			case "web":
				me.Labels["convox.port.443.protocol"] = "tls"

				me.Ports = append(me.Ports, manifest.Port{
					Balancer:  80,
					Container: 4001,
					Public:    true,
				})
				me.Ports = append(me.Ports, manifest.Port{
					Balancer:  443,
					Container: 4001,
					Public:    true,
				})

				me.Environment["PORT"] = "4001"
			}

			m.Services[e.Name] = me
		}

		data, err := yaml.Marshal(m)
		if err != nil {
			return err
		}

		// write the generated docker-compose.yml and return
		return writeFile("docker-compose.yml", data, 0644)
	}

	// write the default if we get here
	return writeAsset("docker-compose.yml", def, nil)
}

type ProcfileEntry struct {
	Name    string
	Command string
}

type Procfile []ProcfileEntry

var reProcfile = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func readProcfile(path string) (Procfile, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

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

	return pf, nil
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	fmt.Printf("Writing %s... ", path)

	if exists(path) {
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

func writeAsset(path, templateName string, input interface{}) error {
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
