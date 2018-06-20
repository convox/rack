package manifest_test

import (
	"testing"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/require"
)

func TestManifestLoad(t *testing.T) {
	n := &manifest.Manifest{
		Environment: manifest.Environment{
			"DEVELOPMENT=true",
			"GLOBAL=true",
			"OTHERGLOBAL",
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
					"DEFAULT=test",
					"DEVELOPMENT=false",
					"SECRET",
				},
				Health: manifest.ServiceHealth{
					Grace:    10,
					Path:     "/",
					Interval: 10,
					Timeout:  9,
				},
				Port:      manifest.ServicePort{Port: 1000, Scheme: "http"},
				Resources: []string{"database"},
				Scale: manifest.ServiceScale{
					Count:  manifest.ServiceScaleCount{Min: 3, Max: 10},
					Cpu:    256,
					Memory: 512,
				},
				Sticky: true,
				Test:   "make  test",
			},
			manifest.Service{
				Name:    "proxy",
				Command: "bash",
				Domains: []string{"bar.example.org", "*.example.org"},
				Health: manifest.ServiceHealth{
					Grace:    5,
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
					Count:  manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    512,
					Memory: 1024,
				},
				Sticky: true,
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
					Grace:    2,
					Interval: 5,
					Path:     "/",
					Timeout:  3,
				},
				Port: manifest.ServicePort{Port: 3000, Scheme: "https"},
				Scale: manifest.ServiceScale{
					Count:  manifest.ServiceScaleCount{Min: 0, Max: 0},
					Cpu:    256,
					Memory: 512,
				},
				Singleton: true,
				Sticky:    false,
			},
			manifest.Service{
				Name: "bar",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Command: "",
				Health: manifest.ServiceHealth{
					Grace:    5,
					Interval: 5,
					Path:     "/",
					Timeout:  4,
				},
				Scale: manifest.ServiceScale{
					Count:  manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    256,
					Memory: 512,
				},
				Sticky: true,
			},
			manifest.Service{
				Name: "scaler",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Command: "",
				Health: manifest.ServiceHealth{
					Grace:    5,
					Interval: 5,
					Path:     "/",
					Timeout:  4,
				},
				Scale: manifest.ServiceScale{
					Count:  manifest.ServiceScaleCount{Min: 1, Max: 5},
					Cpu:    256,
					Memory: 512,
					Targets: manifest.ServiceScaleTargets{
						Cpu:      50,
						Memory:   75,
						Requests: 200,
						Custom: manifest.ServiceScaleMetrics{
							{
								Aggregate:  "max",
								Dimensions: map[string]string{"QueueName": "testqueue"},
								Namespace:  "AWS/SQS",
								Name:       "ApproximateNumberOfMessagesVisible",
								Value:      float64(200),
							},
						},
					},
				},
				Sticky: true,
			},
			manifest.Service{
				Name:    "inherit",
				Command: "inherit",
				Domains: []string{"bar.example.org", "*.example.org"},
				Health: manifest.ServiceHealth{
					Grace:    5,
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
					Count:  manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    512,
					Memory: 1024,
				},
				Sticky: true,
			},
		},
	}

	attrs := []string{"services.proxy.environment", "services.proxy.port", "services", "services.api.build", "services.inherit.health", "services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.dimensions", "environment", "services.inherit.image", "services.scaler.scale.targets.requests", "services.foo.health", "services.foo.singleton", "services.inherit.scale.cpu", "services.inherit.environment", "services.foo.port.port", "services.scaler.scale.targets", "services.scaler.scale.targets.custom", "services.api.domain", "services.foo.scale", "services.proxy", "services.api.test", "resources.database", "services.scaler.scale.targets.memory", "services.api.build.path", "services.api.scale", "services.foo.command", "services.inherit.port", "services.inherit", "services.inherit.scale", "services.proxy.scale.cpu", "services.api.resources", "services.api.health.interval", "services.proxy.scale.memory", "services.proxy.image", "services.api.port", "services.api.health", "services.scaler.scale.count", "services.inherit.command", "services.api", "services.foo", "resources.database.type", "services.proxy.health", "services.inherit.scale.memory", "services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.aggregate", "services.foo.health.timeout", "services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.dimensions.QueueName", "services.api.environment", "services.api.build.manifest", "services.inherit.domain", "services.scaler.scale", "services.scaler", "services.foo.port", "services.proxy.domain", "services.foo.domain", "services.foo.port.scheme", "services.foo.sticky", "services.proxy.command", "services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible", "services.bar", "resources", "services.proxy.scale", "services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.value", "services.scaler.scale.targets.cpu", "services.foo.health.grace"}
	env := map[string]string{"FOO": "bar", "SECRET": "shh", "OTHERGLOBAL": "test"}

	n.SetAttributes(attrs)
	n.SetEnv(env)

	// env processing that normally happens as part of load
	require.NoError(t, n.CombineEnv())
	require.NoError(t, n.ValidateEnv())

	m, err := testdataManifest("full", env)
	require.NoError(t, err)
	require.Equal(t, n, m)

	senv, err := m.ServiceEnvironment("api")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"DEFAULT": "test", "DEVELOPMENT": "false", "GLOBAL": "true", "OTHERGLOBAL": "test", "SECRET": "shh"}, senv)

	s1, err := m.Service("api")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"DEFAULT": "test", "DEVELOPMENT": "false", "GLOBAL": "true"}, s1.EnvironmentDefaults())
	require.Equal(t, "DEFAULT,DEVELOPMENT,GLOBAL,OTHERGLOBAL,SECRET", s1.EnvironmentKeys())

	s2, err := m.Service("proxy")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"DEVELOPMENT": "true", "GLOBAL": "true"}, s2.EnvironmentDefaults())
	require.Equal(t, "DEVELOPMENT,GLOBAL,OTHERGLOBAL,SECRET", s2.EnvironmentKeys())
}

func TestManifestLoadSimple(t *testing.T) {
	_, err := testdataManifest("simple", map[string]string{})
	require.EqualError(t, err, "required env: REQUIRED")

	n := &manifest.Manifest{
		Services: manifest.Services{
			manifest.Service{
				Name: "web",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Environment: manifest.Environment{
					"REQUIRED",
					"DEFAULT=true",
				},
				Health: manifest.ServiceHealth{
					Grace:    5,
					Interval: 5,
					Path:     "/",
					Timeout:  4,
				},
				Scale: manifest.ServiceScale{
					Count:  manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    256,
					Memory: 512,
				},
				Sticky: true,
			},
		},
	}

	n.SetAttributes([]string{"services", "services.web", "services.web.build", "services.web.environment"})
	n.SetEnv(map[string]string{"REQUIRED": "test"})

	// env processing that normally happens as part of load
	require.NoError(t, n.CombineEnv())
	require.NoError(t, n.ValidateEnv())

	m, err := testdataManifest("simple", map[string]string{"REQUIRED": "test"})
	require.NoError(t, err)
	require.Equal(t, n, m)
}

func TestManifestLoadClobberEnv(t *testing.T) {
	env := map[string]string{"FOO": "bar", "REQUIRED": "false"}

	_, err := testdataManifest("simple", env)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"FOO": "bar", "REQUIRED": "false"}, env)
}

func TestManifestLoadInvalid(t *testing.T) {
	m, err := testdataManifest("full", map[string]string{})
	require.Nil(t, m)
	require.Error(t, err, "required env: OTHERGLOBAL, SECRET")

	m, err = testdataManifest("invalid.1", map[string]string{})
	require.Nil(t, m)
	require.Error(t, err, "yaml: line 2: did not find expected comment or line break")

	m, err = testdataManifest("invalid.2", map[string]string{})
	require.NotNil(t, m)
	require.Len(t, m.Services, 0)
}

func testdataManifest(name string, env map[string]string) (*manifest.Manifest, error) {
	data, err := helpers.Testdata(name)
	if err != nil {
		return nil, err
	}

	return manifest.Load(data, env)
}
