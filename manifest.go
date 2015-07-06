package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/convox/build/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Image string `yaml:"image"`
}

func ReadManifest(dir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	var manifest Manifest

	err = yaml.Unmarshal(data, &manifest)

	if err != nil {
		return nil, err
	}

	return &manifest, nil
}
