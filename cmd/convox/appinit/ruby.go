package appinit

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

// RubyApp contains data representing a ruby app
type RubyApp struct {
	af          Appfile
	environment map[string]string
	pf          Procfile
	release     Release
}

// GenerateDockerfile generates a Dockerfile specifically for ruby
func (ra *RubyApp) GenerateDockerfile() ([]byte, error) {
	ra.environment["CURL_CONNECT_TIMEOUT"] = "0" // default timeouts for curl are too aggressive causing failure
	ra.environment["CURL_TIMEOUT"] = "0"
	ra.environment["STACK"] = "cedar-14"

	precompile := `# This is to install sqlite for any ruby apps that need it
# This line can be removed if your app doesn't use sqlite3
RUN apt-get update && apt-get install sqlite3 libsqlite3-dev && apt-get clean`

	input := map[string]interface{}{
		"kind":        "ruby",
		"environment": ra.environment,
		"precompile":  precompile,
	}
	return writeAsset("appinit/templates/Dockerfile", input)
}

// GenerateDockerIgnore generates a .dockerignore file
func (ra *RubyApp) GenerateDockerIgnore() ([]byte, error) {
	input := map[string]interface{}{
		"ignoreFiles": []string{
			"./tmp",
		},
	}
	return writeAsset("appinit/templates/dockerignore", input)
}

// GenerateManifest generates a docker-compose.yml file
func (ra *RubyApp) GenerateManifest() ([]byte, error) {

	m := GenerateManifest(ra.pf, ra.af, ra.release)
	if len(m.Services) == 0 {
		return nil, fmt.Errorf("unable to generate manifest")
	}

	adds := []string{}
	if appFound {
		adds = append(adds, ra.af.Addons...)
	} else {
		adds = append(adds, ra.release.Addons...)
	}
	ParseAddons(adds, &m)

	return yaml.Marshal(m)
}

// Setup runs the buildpacks and collects metadata
// Must be called before other Generate* methods
func (ra *RubyApp) Setup(dir string) error {

	so, err := setup(dir)
	ra.af = so.af
	ra.pf = so.pf
	ra.release = so.release

	ra.environment, err = parseProfiled(so.profile)
	if err != nil {
		fmt.Errorf("parse profiled: %s", err)
	}

	// we do not want all of the buildpack's default processes on convox
	for key := range ra.release.ProcessTypes {
		if key != "web" && key != "database" {
			delete(ra.release.ProcessTypes, key)
		}
	}

	return nil
}
