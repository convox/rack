package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/cmd/convox/templates"
	"gopkg.in/yaml.v2"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "init",
		Description: "initialize an app for local development",
		Usage:       "[directory]",
		Action:      cmdInit,
	})
}

func cmdInit(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	// TODO parse the Dockerfile and build a docker-compose.yml
	if exists("docker-compose.yml") {
		stdcli.Error(fmt.Errorf("Cannot initialize a project that already contains a Dockerfile or docker-compose.yml"))
		return
	}

	err = initApplication(dir)

	if err != nil {
		stdcli.Error(err)
		return
	}
}

func detectApplication(dir string) string {
	switch {
	// case exists(filepath.Join(dir, ".meteor")):
	//   return "meteor"
	// case exists(filepath.Join(dir, "package.json")):
	//   return "node"
	case exists(filepath.Join(dir, "config/application.rb")):
		return "rails"
	case exists(filepath.Join(dir, "config.ru")):
		return "sinatra"
	case exists(filepath.Join(dir, "Gemfile.lock")):
		return "ruby"
	}

	return "unknown"
}

func initApplication(dir string) error {
	// TODO parse the Dockerfile and build a docker-compose.yml
	if exists("docker-compose.yml") {
		return nil
	}

	kind := detectApplication(dir)

	fmt.Printf("Initializing %s\n", kind)

	if err := writeAsset("Dockerfile", fmt.Sprintf("init/%s/Dockerfile", kind)); err != nil {
		return err
	}

	if err := generateManifest(dir, fmt.Sprintf("init/%s/docker-compose.yml", kind)); err != nil {
		return err
	}

	if err := writeAsset(".dockerignore", fmt.Sprintf("init/%s/.dockerignore", kind)); err != nil {
		return err
	}

	return nil
}

type ManifestEntry struct {
	Build   string   `yaml:"build,omitempty"`
	Command string   `yaml:"command,omitempty"`
	Labels  []string `yaml:"labels,omitempty"`
	Ports   []string `yaml:"ports,omitempty"`
}

type Manifest map[string]ManifestEntry

func generateManifest(dir string, def string) error {
	if exists("Procfile") {
		pf, err := readProcfile("Procfile")

		if err != nil {
			return err
		}

		m := Manifest{}

		for _, e := range pf {
			me := ManifestEntry{
				Build:   ".",
				Command: e.Command,
			}

			switch e.Name {
			case "web":
				me.Labels = append(me.Labels, "convox.port.443.protocol=tls")
				me.Labels = append(me.Labels, "convox.port.443.proxy=true")

				me.Ports = append(me.Ports, "80:4000")
				me.Ports = append(me.Ports, "443:4001")
			}

			m[e.Name] = me
		}

		data, err := yaml.Marshal(m)

		if err != nil {
			return err
		}

		// write the generated docker-compose.yml and return
		return writeFile("docker-compose.yml", data, 0644)
	}

	// write the default if we get here
	return writeAsset("docker-compose.yml", def)
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

func writeAsset(path, template string) error {
	data, err := templates.Asset(template)

	if err != nil {
		return err
	}

	info, err := templates.AssetInfo(template)

	if err != nil {
		return err
	}

	return writeFile(path, data, info.Mode())
}
