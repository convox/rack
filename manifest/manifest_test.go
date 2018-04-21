package manifest_test

import (
	"testing"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

func TestManifestLoad(t *testing.T) {
	m, err := testdataManifest("full", manifest.Environment{"FOO": "bar", "SECRET": "shh"})
	if !assert.NoError(t, err) {
		return
	}

	n := &manifest.Manifest{
		Environment: manifest.Environment{
			"DEVELOPMENT": "false",
			"SECRET":      "shh",
		},
		Resources: manifest.Resources{
			manifest.Resource{
				Name: "database",
				Type: "postgres",
			},
		},
		Services: manifest.Services{
			manifest.Service{
				Name: "api",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile2",
					Path:     "api",
				},
				Domains: []string{"foo.example.org"},
				Command: "",
				Environment: []string{
					"DEVELOPMENT=false",
					"SECRET",
				},
				Health: manifest.ServiceHealth{
					Path:     "/",
					Interval: 10,
					Timeout:  9,
				},
				Port:      manifest.ServicePort{Port: 1000, Scheme: "http"},
				Resources: []string{"database"},
				Scale: manifest.ServiceScale{
					Count:  &manifest.ServiceScaleCount{Min: 3, Max: 10},
					Cpu:    256,
					Memory: 512,
				},
				Test: "make  test",
			},
			manifest.Service{
				Name:    "proxy",
				Command: "bash",
				Domains: []string{"bar.example.org", "*.example.org"},
				Health: manifest.ServiceHealth{
					Path:     "/auth",
					Interval: 5,
					Timeout:  4,
				},
				Image: "ubuntu:16.04",
				Environment: []string{
					"SECRET",
				},
				Port: manifest.ServicePort{Port: 2000, Scheme: "https"},
				Scale: manifest.ServiceScale{
					Count:  &manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    512,
					Memory: 1024,
				},
			},
			manifest.Service{
				Name: "foo",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Command: "foo",
				Domains: []string{"baz.example.org", "qux.example.org"},
				Health: manifest.ServiceHealth{
					Interval: 5,
					Path:     "/",
					Timeout:  3,
				},
				Port: manifest.ServicePort{Port: 3000, Scheme: "https"},
				Scale: manifest.ServiceScale{
					Count:  &manifest.ServiceScaleCount{Min: 0, Max: 0},
					Cpu:    256,
					Memory: 512,
				},
			},
			manifest.Service{
				Name: "bar",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Command: "",
				Health: manifest.ServiceHealth{
					Interval: 5,
					Path:     "/",
					Timeout:  4,
				},
				Scale: manifest.ServiceScale{
					Count:  &manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    256,
					Memory: 512,
				},
			},
			manifest.Service{
				Name: "scaler",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Command: "",
				Health: manifest.ServiceHealth{
					Interval: 5,
					Path:     "/",
					Timeout:  4,
				},
				Scale: manifest.ServiceScale{
					Count:  &manifest.ServiceScaleCount{Min: 1, Max: 5},
					Cpu:    256,
					Memory: 512,
					Targets: manifest.ServiceScaleTargets{
						Cpu:      50,
						Memory:   75,
						Requests: 200,
					},
				},
			},
		},
	}

	assert.Equal(t, n, m)
}

func TestManifestLoadInvalid(t *testing.T) {
	m, err := testdataManifest("invalid.1", manifest.Environment{})
	assert.Nil(t, m)
	assert.Error(t, err, "yaml: line 2: did not find expected comment or line break")

	m, err = testdataManifest("invalid.2", manifest.Environment{})
	assert.NotNil(t, m)
	assert.Len(t, m.Services, 0)
}

func testdataManifest(name string, env manifest.Environment) (*manifest.Manifest, error) {
	data, err := helpers.Testdata(name)
	if err != nil {
		return nil, err
	}

	return manifest.Load(data, env)
}
