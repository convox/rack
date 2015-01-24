package provider

import (
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Manifest struct {
	Processes []ManifestProcess
	Resources []ManifestResource
}

type ManifestEntry struct {
	Build string   `yaml:"build"`
	Env   []string `yaml:"env"`
	Image string   `yaml:"image"`
	Links []string `yaml:"links"`
}

type ManifestResource struct {
	Name    string
	Service string
}

type ManifestProcess struct {
	Env   []string
	Name  string
	Links []string
}

func NewManifest(raw string) (*Manifest, error) {
	var mr map[string]ManifestEntry

	err := yaml.Unmarshal([]byte(raw), &mr)

	if err != nil {
		return nil, err
	}

	manifest := &Manifest{}

	for name, entry := range mr {
		if strings.HasPrefix(entry.Image, "convox/") {
			manifest.Resources = append(manifest.Resources, ManifestResource{Name: name, Service: entry.Image[7:]})
		} else {
			manifest.Processes = append(manifest.Processes, ManifestProcess{Name: name, Env: entry.Env, Links: entry.Links})
		}
	}

	return manifest, nil
}

type ManifestResourceParams struct {
	Cidr string
	Name string
	Vpc  string
}

func (m *Manifest) Generate(vpc, cidr string) (string, error) {
}

func (mp *ManifestProcess) GenerateFormation(vpc, cidr string) (string, error) {
	return buildResourceTemplate("process", "formation", ManifestProcessParams{})
}

func (mp *ManifestProcess) GenerateOutputs(vpc, cidr string) (string, error) {
	return buildResourceTemplate("process", "outputs", ManifestProcessParams{})
}

func (mr *ManifestResource) GenerateEnv(vpc, cidr string) (string, error) {
	return buildResourceTemplate(mr.Service, "env", ManifestResourceParams{Name: mr.Name, Cidr: cidr, Vpc: vpc})
}

func (mr *ManifestResource) GenerateFormation(vpc, cidr string) (string, error) {
	return buildResourceTemplate(mr.Service, "formation", ManifestResourceParams{Name: mr.Name, Cidr: cidr, Vpc: vpc})
}
