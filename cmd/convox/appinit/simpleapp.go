package appinit

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

// SimpleApp contains data representing a generic app
type SimpleApp struct {
	Kind string

	af          Appfile
	environment map[string]string
	pf          Procfile
	release     Release
}

// GenerateDockerfile generates a Dockerfile
func (sa *SimpleApp) GenerateDockerfile() ([]byte, error) {
	input := map[string]interface{}{
		"kind":        sa.Kind,
		"environment": sa.environment,
	}
	return writeAsset("appinit/templates/Dockerfile", input)
}

// GenerateDockerIgnore generates a .dockerignore file
func (sa *SimpleApp) GenerateDockerIgnore() ([]byte, error) {
	input := map[string]interface{}{
		"ignoreFiles": []string{"./.heroku"},
	}
	return writeAsset("appinit/templates/dockerignore", input)
}

// GenerateLocalEnv generates a .env file
func (sa *SimpleApp) GenerateLocalEnv() ([]byte, error) {
	return nil, nil
}

// GenerateGitIgnore generates a .gitignore file
func (sa *SimpleApp) GenerateGitIgnore() ([]byte, error) {
	return writeAsset("appinit/templates/gitignore", nil)
}

// GenerateManifest generates a docker-compose.yml file
func (sa *SimpleApp) GenerateManifest() ([]byte, error) {
	m := GenerateManifest(sa.pf, sa.af, sa.release)
	if len(m.Services) == 0 {
		return nil, fmt.Errorf("unable to generate manifest")
	}

	adds := []string{}
	if appFound {
		adds = append(adds, sa.af.Addons...)
	} else {
		adds = append(adds, sa.release.Addons...)
	}
	ParseAddons(adds, &m)

	return yaml.Marshal(m)
}

// Setup runs the buildpacks and collects metadata
// Must be called before other Generate* methods
func (sa *SimpleApp) Setup(dir string) error {
	so, err := setup(dir)
	sa.af = so.af
	sa.pf = so.pf
	sa.release = so.release

	sa.environment, err = parseProfiled(so.profile)
	if err != nil {
		fmt.Errorf("parse profiled: %s", err)
	}
	return nil
}
