package models_test

import (
	"os"
	"testing"

	"github.com/convox/rack/api/models"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox-test")
}

func TestRackStackName(t *testing.T) {
	r := models.App{
		Name: "convox-test",
	}

	assert.Equal(t, "convox-test", r.StackName())
}

func TestAppStackName(t *testing.T) {
	// unbound app (no rack prefix)
	a := models.App{
		Name: "httpd",
		Tags: map[string]string{
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}

	assert.Equal(t, "httpd", a.StackName())

	// bound app (rack prefix, and Name tag)
	a = models.App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}

	assert.Equal(t, "convox-test-httpd", a.StackName())
}
