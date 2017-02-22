package appinit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"

	"github.com/convox/rack/cmd/convox/appinit/templates"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/manifest"
)

var dockerBin = helpers.DetectDocker()

type AppFramework interface {
	GenerateEntrypoint() ([]byte, error)
	GenerateDockerfile() ([]byte, error)
	GenerateDockerIgnore() ([]byte, error)
	GenerateManifest() ([]byte, error)
	Setup(string) error
}

// EnvEntry is an environment entry from an app.json file
type EnvEntry struct {
	Value string
}

// Appfile represent specific fields of an app.json file
type Appfile struct {
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
	ConfigVars   map[string]string `yaml:"config_vars"`
	ProcessTypes map[string]string `yaml:"default_process_types"`
}

// ReadProcfileData reads data that follows the Procfile format
func ReadProcfileData(data []byte) Procfile {
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

// ReadProcfile reads a file and returns an Procfile
func ReadProcfile(path string) (Procfile, error) {
	if !helpers.Exists(path) {
		return Procfile{}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ReadProcfileData(data), nil
}

// ReadAppfileData reads data that follows the app.json manifest format
func ReadAppfileData(data []byte) (Appfile, error) {
	af := Appfile{
		[]string{},
		nil,
	}

	if err := json.Unmarshal(data, &af); err != nil {
		return af, err
	}

	return af, nil
}

// ReadAppfile reads a file and returns an Appfile
func ReadAppfile(path string) (Appfile, error) {
	var af Appfile

	if helpers.Exists(path) {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return af, err
		}

		af, err = ReadAppfileData(data)
		if err != nil {
			return af, err
		}
		appFound = true
	}

	return af, nil
}

// GenerateManifest generates a Manifest from the union of a Procfile, Appfile and Release data
func GenerateManifest(pf Procfile, af Appfile, r Release) manifest.Manifest {

	m := manifest.Manifest{
		Services: make(map[string]manifest.Service),
		Version:  "2",
	}

	// No Procfile, rely on default release data
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

		if me, ok := m.Services["web"]; ok {
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

			m.Services["web"] = me
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
		}
		m.Services[e.Name] = me
	}

	for k, v := range r.ConfigVars {
		for name, s := range m.Services {
			s.Environment = append(s.Environment, manifest.EnvironmentItem{
				Name:  k,
				Value: v,
			})
			m.Services[name] = s
		}
	}

	for k, v := range af.Env {
		for name, s := range m.Services {
			s.Environment = append(s.Environment, manifest.EnvironmentItem{
				Name:  k,
				Value: v.Value,
			})
			m.Services[name] = s
		}
	}

	// Some buildpacks are known to use environment variables in the command string so we escape them
	for name, s := range m.Services {
		for _, env := range s.Environment {
			if s.Command.String != "" {
				if strings.Contains(s.Command.String, fmt.Sprintf("$%s", env.Name)) {
					s.Command.String = strings.Replace(
						s.Command.String,
						fmt.Sprintf("$%s", env.Name),
						fmt.Sprintf("$$%s", env.Name),
						-1,
					)
					m.Services[name] = s
				}
			}
		}
	}

	return m
}

// generateManifestData creates a Manifest from files in the directory
func generateManifestData(dir string) ([]byte, error) {
	pf, err := ReadProcfile(path.Join(dir, "Procfile"))
	if err != nil {
		return nil, err
	}

	am, err := ReadAppfile(path.Join(dir, "app.json"))
	if err != nil {
		return nil, err
	}

	var release Release
	if len(pf) == 0 || !appFound {
		args := []string{"run", "--rm", "-v", fmt.Sprintf("%s:/app", dir), "convox/init"}

		r, err := exec.Command(dockerBin, append(args, "release")...).Output()
		if err != nil {
			return nil, err
		}

		fmt.Printf("r = %s\n", string(r))

		if err := yaml.Unmarshal(r, &release); err != nil {
			return nil, err
		}
	}

	m := GenerateManifest(pf, am, release)
	if len(m.Services) == 0 {
		return nil, fmt.Errorf("unable to generate manifest")
	}

	adds := []string{}
	if appFound {
		adds = append(adds, am.Addons...)
	} else {
		adds = append(adds, release.Addons...)
	}
	ParseAddons(adds, &m)

	return yaml.Marshal(m)
}

// writeAsset is a helper function that generates an asset
func writeAsset(templateName string, input map[string]interface{}) ([]byte, error) {
	data, err := templates.Asset(templateName)
	if err != nil {
		return nil, err
	}

	if input != nil {
		tmpl, err := template.New(templateName).Parse(string(data))
		if err != nil {
			return nil, err
		}

		var formation bytes.Buffer

		err = tmpl.Execute(&formation, input)
		if err != nil {
			return nil, err
		}

		data = formation.Bytes()
	}

	return data, nil
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
