package manifest_test

import (
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
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
				Options: map[string]string{
					"size": "db.t2.large",
				},
			},
		},
		Services: manifest.Services{
			manifest.Service{
				Name: "api",
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile2",
					Path:     "api",
				},
				Command: "",
				Domains: []string{"foo.example.org"},
				Drain:   30,
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
				Init:      true,
				Port:      manifest.ServicePort{Port: 1000, Protocol: "tcp", Scheme: "http"},
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
				Drain:   30,
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
				Port: manifest.ServicePort{Port: 2000, Protocol: "tcp", Scheme: "https"},
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
				Drain:   60,
				Health: manifest.ServiceHealth{
					Grace:    2,
					Interval: 5,
					Path:     "/",
					Timeout:  3,
				},
				Port: manifest.ServicePort{Port: 3000, Protocol: "tcp", Scheme: "https"},
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
				Drain:   30,
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
				Drain:   30,
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
				Drain:   30,
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
				Port: manifest.ServicePort{Port: 2000, Protocol: "tcp", Scheme: "https"},
				Scale: manifest.ServiceScale{
					Count:  manifest.ServiceScaleCount{Min: 1, Max: 1},
					Cpu:    512,
					Memory: 1024,
				},
				Sticky: true,
			},
			manifest.Service{
				Name: "agent",
				Agent: manifest.ServiceAgent{
					Enabled: true,
					Ports: []manifest.ServicePort{
						{Port: 5000, Protocol: "udp", Scheme: "http"},
						{Port: 5001, Protocol: "tcp", Scheme: "http"},
						{Port: 5002, Protocol: "tcp", Scheme: "http"},
					},
				},
				Build: manifest.ServiceBuild{
					Manifest: "Dockerfile",
					Path:     ".",
				},
				Drain: 30,
				Health: manifest.ServiceHealth{
					Grace:    5,
					Path:     "/",
					Interval: 5,
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

	attrs := []string{"services.proxy.environment",
		"environment",
		"resources",
		"resources.database",
		"resources.database.options",
		"resources.database.options.size",
		"resources.database.type",
		"services",
		"services.agent",
		"services.agent.agent",
		"services.agent.agent.ports",
		"services.api",
		"services.api.build",
		"services.api.build.manifest",
		"services.api.build.path",
		"services.api.domain",
		"services.api.environment",
		"services.api.health",
		"services.api.health.interval",
		"services.api.init",
		"services.api.port",
		"services.api.resources",
		"services.api.scale",
		"services.api.test",
		"services.bar",
		"services.foo",
		"services.foo.command",
		"services.foo.domain",
		"services.foo.drain",
		"services.foo.health",
		"services.foo.health.grace",
		"services.foo.health.timeout",
		"services.foo.port",
		"services.foo.port.port",
		"services.foo.port.scheme",
		"services.foo.scale",
		"services.foo.singleton",
		"services.foo.sticky",
		"services.inherit",
		"services.inherit.command",
		"services.inherit.domain",
		"services.inherit.environment",
		"services.inherit.health",
		"services.inherit.image",
		"services.inherit.port",
		"services.inherit.scale",
		"services.inherit.scale.cpu",
		"services.inherit.scale.memory",
		"services.proxy",
		"services.proxy.command",
		"services.proxy.domain",
		"services.proxy.health",
		"services.proxy.image",
		"services.proxy.port",
		"services.proxy.scale",
		"services.proxy.scale.cpu",
		"services.proxy.scale.memory",
		"services.scaler",
		"services.scaler.scale",
		"services.scaler.scale.count",
		"services.scaler.scale.targets",
		"services.scaler.scale.targets.cpu",
		"services.scaler.scale.targets.custom",
		"services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible",
		"services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.aggregate",
		"services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.dimensions",
		"services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.dimensions.QueueName",
		"services.scaler.scale.targets.custom.AWS/SQS/ApproximateNumberOfMessagesVisible.value",
		"services.scaler.scale.targets.memory",
		"services.scaler.scale.targets.requests",
	}

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
				Drain: 30,
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

func TestManifestEnvManipulation(t *testing.T) {
	m, err := testdataManifest("env", map[string]string{})
	require.NotNil(t, m)
	require.NoError(t, err)

	require.Equal(t, "train-intent", m.Services[0].EnvironmentDefaults()["QUEUE_NAME"])
	require.Equal(t, "delete-intent", m.Services[1].EnvironmentDefaults()["QUEUE_NAME"])
}

func testdataManifest(name string, env map[string]string) (*manifest.Manifest, error) {
	data, err := helpers.Testdata(name)
	if err != nil {
		return nil, err
	}

	return manifest.Load(data, env)
}
