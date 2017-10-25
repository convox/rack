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
	"github.com/convox/rack/manifest1"
)

var dockerBin = helpers.DetectDocker()

type AppFramework interface {
	GenerateDockerfile() ([]byte, error)
	GenerateDockerIgnore() ([]byte, error)
	GenerateLocalEnv() ([]byte, error)
	GenerateGitIgnore() ([]byte, error)
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
func GenerateManifest(pf Procfile, af Appfile, r Release) manifest1.Manifest {

	m := manifest1.Manifest{
		Services: make(map[string]manifest1.Service),
		Version:  "2",
	}

	// No Procfile, rely on default release data
	if len(pf) == 0 {
		for name, cmd := range r.ProcessTypes {
			me := manifest1.Service{
				Build: manifest1.Build{
					Context: ".",
				},
				Command: manifest1.Command{
					String: cmd,
				},
			}

			m.Services[name] = me
		}

		if me, ok := m.Services["web"]; ok {
			me.Ports = append(me.Ports, manifest1.Port{
				Name:      "80",
				Balancer:  80,
				Container: 4001,
				Public:    true,
				Protocol:  manifest1.TCP,
			})
			me.Ports = append(me.Ports, manifest1.Port{
				Name:      "443",
				Balancer:  443,
				Container: 4001,
				Public:    true,
				Protocol:  manifest1.TCP,
			})

			me.Environment = append(me.Environment, manifest1.EnvironmentItem{
				Name:  "PORT",
				Value: "4001",
			})

			m.Services["web"] = me
		}
	}

	for _, e := range pf {
		me := manifest1.Service{
			Build: manifest1.Build{
				Context: ".",
			},
			Command: manifest1.Command{
				String: e.Command,
			},
			Environment: manifest1.Environment{},
			Labels:      make(manifest1.Labels),
			Ports:       make(manifest1.Ports, 0),
		}

		switch e.Name {
		case "web":
			me.Name = "web"
			me.Labels["convox.port.443.protocol"] = "tls"

			me.Ports = append(me.Ports, manifest1.Port{
				Name:      "80",
				Balancer:  80,
				Container: 4001,
				Public:    true,
				Protocol:  manifest1.TCP,
			})
			me.Ports = append(me.Ports, manifest1.Port{
				Name:      "443",
				Balancer:  443,
				Container: 4001,
				Public:    true,
				Protocol:  manifest1.TCP,
			})

			me.Environment = append(me.Environment, manifest1.EnvironmentItem{
				Name:  "PORT",
				Value: "4001",
			})
		}
		m.Services[e.Name] = me
	}

	for k, v := range r.ConfigVars {
		for name, s := range m.Services {
			s.Environment = append(s.Environment, manifest1.EnvironmentItem{
				Name:  k,
				Value: v,
			})
			m.Services[name] = s
		}
	}

	for k, v := range af.Env {
		for name, s := range m.Services {
			s.Environment = append(s.Environment, manifest1.EnvironmentItem{
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
type AddonHandler func(m *manifest1.Manifest)

// ParseAddons iterates through an apps addons and edits the manifest accordingly
func ParseAddons(addons []string, m *manifest1.Manifest) {
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

// postgresAddon configures a Manifest to have a postgres service
func postgresAddon(m *manifest1.Manifest) {
	s := manifest1.Service{
		Image: "convox/postgres",
		Name:  "database",
		Ports: manifest1.Ports{
			{
				Balancer:  5432,
				Container: 5432,
				Public:    false,
				Protocol:  manifest1.TCP,
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

// setupOutput is a container type that holds all the data collected by setup()
type setupOutput struct {
	af      Appfile
	pf      Procfile
	release Release
	profile []byte
}

// setup is a common helper function to gather buildpack metadata
func setup(dir string) (setupOutput, error) {
	var err error

	so := setupOutput{}

	so.pf, err = ReadProcfile(path.Join(dir, "Procfile"))
	if err != nil {
		return so, err
	}

	so.af, err = ReadAppfile(path.Join(dir, "app.json"))
	if err != nil {
		return so, err
	}

	// We start a container with tailing nothing to keep it running and work inside it
	args := []string{"run", "--rm", "-d",
		"-v", fmt.Sprintf("%s:/app", dir),
		"convox/init", "tail", "-f", "/dev/null",
	}

	output, err := exec.Command(dockerBin, args...).CombinedOutput()
	if err != nil {
		return so, fmt.Errorf("buildpack contaier: %s - %s", string(output), err)
	}

	containerID := strings.TrimSpace(string(output))
	defer exec.Command(dockerBin, "rm", "--force", containerID).Run()

	args = []string{"exec", containerID, "compile-release"}
	r, err := exec.Command(dockerBin, args...).CombinedOutput()
	if err != nil {

		// output could be huge and not user friendly as a wall of red text if an error type
		fmt.Println(string(r))
		return so, fmt.Errorf("buildpack compile: %s", err)
	}

	if err := yaml.Unmarshal(r, &so.release); err != nil {
		return so, fmt.Errorf("buildpack release: %s", err)
	}

	args = []string{"exec", containerID, "profiled"}
	so.profile, err = exec.Command(dockerBin, args...).CombinedOutput()
	if err != nil {
		return so, fmt.Errorf("buildpack profile: %s - %s", string(so.profile), err)
	}

	return so, nil
}

// parseProfiled is a common helper function to gather env vars
// from files contained in an apps profile.d directory.
func parseProfiled(data []byte) (map[string]string, error) {
	env := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		// we only care about lines that start with export
		if !strings.HasPrefix(line, "export") {
			continue
		}

		l := strings.SplitN(line, "=", 2)
		if len(l) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.Replace(l[0], "export", "", -1))

		env[key] = l[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}
