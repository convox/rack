package appify

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v2"

	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/manifest"
)

// Buildpack type representing a Heroku Buildpack
type Buildpack struct {
	manifest  manifest.Manifest
	setup     bool
	tmplInput map[string]interface{}
}

// ProcfileEntry is an entry in a Procfile
type ProcfileEntry struct {
	Name    string
	Command string
}

// Procfile represents a Procfile used in Heroku-based apps
type Procfile []ProcfileEntry

var reProcfile = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

// Appfiy generates the files needed for an app
// Must be called after Setup()
func (bp *Buildpack) Appify() error {
	if !bp.setup {
		return fmt.Errorf("must call Setup() first")
	}

	if err := writeAsset("entrypoint.sh", "appify/templates/buildpack/entrypoint.sh", nil); err != nil {
		return err
	}

	if err := writeAsset("Dockerfile", "appify/templates/buildpack/Dockerfile", bp.tmplInput); err != nil {
		return err
	}

	if err := writeAsset(".dockerignore", "appify/templates/buildpack/.dockerignore", nil); err != nil {
		return err
	}

	data, err := yaml.Marshal(bp.manifest)
	if err != nil {
		return err
	}

	return writeFile("docker-compose.yml", data, 0644)
}

// Setup reads the location for Buildpack specifc files and settings
func (bp *Buildpack) Setup(location string) error {
	kind := detectBuildpack(location)
	if kind == "unknown" {
		// TODO: track this event
		return fmt.Errorf("unknown Buildpack type")
	}

	bp.tmplInput = map[string]interface{}{
		"framework":   kind,
		"environment": buildpackEnvironment(kind),
	}

	proc, err := readProcfile("Procfile")
	if err != nil {
		return fmt.Errorf("reading Procfile : %s", err)
	}

	bp.manifest = generateManifest(proc)
	bp.setup = true

	return nil
}

func detectBuildpack(location string) string {
	switch {
	case helpers.Exists(filepath.Join(location, "requirements.txt")) || helpers.Exists(filepath.Join(location, "setup.py")):
		return "python"
	case helpers.Exists(filepath.Join(location, "package.json")):
		return "nodejs"
	case helpers.Exists(filepath.Join(location, "Gemfile")):
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

// generateManifest generates a Manifest from a Procfile
func generateManifest(proc Procfile) manifest.Manifest {

	m := manifest.Manifest{
		Services: make(map[string]manifest.Service),
	}

	for _, e := range proc {
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

	return m

	//data, err := yaml.Marshal(m)
	//if err != nil {
	//return err
	//}

	// write the generated docker-compose.yml and return
	//return writeFile("docker-compose.yml", data, 0644)
}
