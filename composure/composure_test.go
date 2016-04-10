package composure

import (
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/composure/provider"
	"github.com/convox/rack/composure/structs"
)

func TestNil(t *testing.T) {
	assert.Nil(t, nil)
}

// TestArchive asserts the order of interface calls to:
//  Read a project and manifest from an app directory
//  Pull, build and tag images
//  Tag, and push images to a remote registry and repository
func TestArchive(t *testing.T) {
	// set current provider
	testProvider := &provider.TestProviderRunner{}
	provider.CurrentProvider = testProvider

	// Build() with a project directory and docker-compose.yml manifest
	testProvider.On("ManifestRun", ".", "docker-compose.yml")
	testProvider.On("ProjectName", ".").Return("myapp", nil)
	testProvider.On("ManifestLoad", ".", "docker-compose.yml").Return(&structs.Manifest{
		"web": structs.ManifestEntry{
			Image: "httpd",
		},
		"worker": structs.ManifestEntry{
			Build:      ".",
			Dockerfile: "Dockerfile",
		},
		"worker2": structs.ManifestEntry{
			Build:      ".",
			Dockerfile: "Dockerfile-worker2",
		},
	}, nil)

	// pull, build and tag all images
	testProvider.On("ManifestBuild", ".", "docker-compose.yml").Return(map[string]string{}, nil)
	testProvider.On("ImagePull", "httpd").Return(nil)
	testProvider.On("ImageTag", "httpd", "myapp/web").Return(nil)

	testProvider.On("ImageBuild", ".", "Dockerfile", "convox-xvlbzgbaic").Return(nil)
	testProvider.On("ImageTag", "convox-xvlbzgbaic", "myapp/worker").Return(nil)

	testProvider.On("ImageBuild", ".", "Dockerfile-worker2", "convox-mrajwwhthc").Return(nil)
	testProvider.On("ImageTag", "convox-mrajwwhthc", "myapp/worker2").Return(nil)

	// push all images
	testProvider.On("ManifestPush", ".", "docker-compose.yml", "132866487567.dkr.ecr.us-east-1.amazonaws.com", "convox-myapp-owdnefujkr").Return(nil)
	testProvider.On("ImagePush", "myapp/web", "132866487567.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-owdnefujkr:web").Return(nil)
	testProvider.On("ImagePush", "myapp/worker", "132866487567.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-owdnefujkr:worker").Return(nil)
	testProvider.On("ImagePush", "myapp/worker2", "132866487567.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-owdnefujkr:worker2").Return(nil)

	err := Archive(".", "docker-compose.yml", "132866487567.dkr.ecr.us-east-1.amazonaws.com", "convox-myapp-owdnefujkr")
	assert.Nil(t, err)
}

// TestStart asserts the order of interface calls to:
//  Read a project and manifest from an app directory
//  Pull, build and tag images
//  Introspect manifest, network and images to determine LINK_ environment
//  Run processes with the correct image, command and environment options
func TestStart(t *testing.T) {
	// set current provider
	testProvider := &provider.TestProviderRunner{}
	provider.CurrentProvider = testProvider

	// Start() with a project directory and docker-compose.yml manifest
	testProvider.On("ManifestRun", ".", "docker-compose.yml")
	testProvider.On("ProjectName", ".").Return("myapp", nil)
	testProvider.On("ManifestLoad", ".", "docker-compose.yml").Return(&structs.Manifest{
		"web": structs.ManifestEntry{
			Image:       "httpd",
			Environment: []string{"REDIS_URL"},
			Links:       []string{"redis"},
			Ports:       []string{"80:80"},
		},
		"worker": structs.ManifestEntry{
			Build:       ".",
			Dockerfile:  "Dockerfile-worker",
			Command:     []string{"node", "worker.js"},
			Environment: []string{"REDIS_URL", "MAX_QUEUE_DEPTH=10"},
			Links:       []string{"redis"},
		},
		"redis": structs.ManifestEntry{
			Image:       "convox/redis",
			Environment: []string{"LINK_PATH=/1"}, // demonstrate overriding or filling in for a missing LINK_PATH on convox/redis
			Ports:       []string{"6379"},
		},
	}, nil)

	// pull, build and tag all images
	testProvider.On("ManifestBuild", ".", "docker-compose.yml").Return(map[string]string{}, nil)
	testProvider.On("ImagePull", "httpd").Return(nil)
	testProvider.On("ImageTag", "httpd", "myapp/web").Return(nil)

	testProvider.On("ImagePull", "convox/redis").Return(nil)
	testProvider.On("ImageTag", "convox/redis", "myapp/redis").Return(nil)

	testProvider.On("ImageBuild", ".", "Dockerfile-worker", "convox-tcuaxhxkqf").Return(nil)
	testProvider.On("ImageTag", "convox-tcuaxhxkqf", "myapp/worker").Return(nil)

	// introspect runtime options
	testProvider.On("NetworkInspect").Return("172.17.0.1", nil)

	testProvider.On("ImageInspect", "myapp/redis").Return(map[string]string{
		"LINK_URL":      "redis://redis:password@172.17.0.1:6379/1",
		"LINK_HOST":     "172.17.0.1",
		"LINK_SCHEME":   "redis",
		"LINK_PORT":     "6379",
		"LINK_USERNAME": "redis",
		"LINK_PASSWORD": "password",
		"LINK_PATH":     "/0",
	}, nil)

	// run containers
	testProvider.On("ProcessRun", "myapp/web", []string{}, "myapp-web", []string{"80:80"}, map[string]string{
		"REDIS_URL":      "redis://redis:password@172.17.0.1:6379/1",
		"REDIS_HOST":     "172.17.0.1",
		"REDIS_SCHEME":   "redis",
		"REDIS_PORT":     "6379",
		"REDIS_USERNAME": "redis",
		"REDIS_PASSWORD": "password",
		"REDIS_PATH":     "/1",
	}).Return(nil)

	testProvider.On("ProcessRun", "myapp/worker", []string{"node", "worker.js"}, "myapp-worker", []string{}, map[string]string{
		"MAX_QUEUE_DEPTH": "10",
		"REDIS_URL":       "redis://redis:password@172.17.0.1:6379/1",
		"REDIS_HOST":      "172.17.0.1",
		"REDIS_SCHEME":    "redis",
		"REDIS_PORT":      "6379",
		"REDIS_USERNAME":  "redis",
		"REDIS_PASSWORD":  "password",
		"REDIS_PATH":      "/1",
	}).Return(nil)

	testProvider.On("ProcessRun", "myapp/redis", []string{}, "myapp-redis", []string{"6379"}, map[string]string{
		"LINK_PATH": "/1",
	}).Return(nil)

	err := Start(".", "docker-compose.yml")
	assert.Nil(t, err)
}
