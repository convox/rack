package appinit

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

type SimpleApp struct {
	Kind string

	af          Appfile
	environment map[string]string
	pf          Procfile
	release     Release
}

func (sa *SimpleApp) GenerateDockerfile() ([]byte, error) {
	input := map[string]interface{}{
		"kind":        sa.Kind,
		"environment": sa.environment,
	}
	return writeAsset("appinit/templates/Dockerfile", input)
}

func (sa *SimpleApp) GenerateDockerIgnore() ([]byte, error) {
	input := map[string]interface{}{
		"ignoreFiles": []string{"./.heroku"},
	}
	return writeAsset("appinit/templates/dockerignore", input)
}

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

func (sa *SimpleApp) Setup(dir string) error {
	so, err := setup(dir)
	sa.af = so.af
	sa.pf = so.pf
	sa.release = so.release

	fmt.Printf("so.profile = %s\n", string(so.profile))

	sa.environment, err = parseProfiled(so.profile)
	if err != nil {
		fmt.Errorf("parse profiled: %s", err)
	}
	return nil
}
