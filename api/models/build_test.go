package models

import (
	"os"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

func init() {
	os.Setenv("AWS_REGION", "test")
	os.Setenv("REGISTRY_HOST", "convox-826133048.us-east-1.elb.amazonaws.com:5000")
}

func TestBuildImagesRegistry(t *testing.T) {
	provider.TestProvider.App = structs.App{
		Name: "bar",
	}

	b := Build{
		App: "bar",
		Manifest: `web:
  image: httpd
`,
		Id: "BSUSBFCUCSA",
	}

	imgs, err := b.Images()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(imgs))
	assert.Equal(t, "convox-826133048.us-east-1.elb.amazonaws.com:5000/bar-web:BSUSBFCUCSA", imgs[0])
}

func TestBuildImagesECR(t *testing.T) {
	provider.TestProvider.App = structs.App{
		Name: "bar",
		Outputs: map[string]string{
			"RegistryId":         "826133048",
			"RegistryRepository": "bar-zridvyqapp",
		},
	}

	b := Build{
		App: "bar",
		Manifest: `web:
  image: httpd
`,
		Id: "BSUSBFCUCSA",
	}

	imgs, err := b.Images()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(imgs))
	assert.Equal(t, "826133048.dkr.ecr.test.amazonaws.com/bar-zridvyqapp:web.BSUSBFCUCSA", imgs[0])
}
