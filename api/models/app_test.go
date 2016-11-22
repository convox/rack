package models_test

import (
	"os"
	"sort"
	"testing"

	"github.com/convox/rack/api/models"
	"github.com/convox/rack/manifest"
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

func TestAppCronJobs(t *testing.T) {

	m := manifest.Manifest{
		Version: "1",
		Services: map[string]manifest.Service{
			"one": {
				Name: "one",
				Labels: manifest.Labels{
					"convox.cron.task1": "00 19 * * ? ls -la",
				},
			},
			"two": {
				Name: "two",
				Labels: manifest.Labels{
					"convox.cron.task2": "00 20 * * ? ls -la",
					"convox.cron.task3": "00 21 * * ? ls -la",
				},
			},
		},
	}

	a := models.App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}

	cj := a.CronJobs(m)
	sort.Sort(models.CronJobs(cj))

	assert.Equal(t, len(cj), 3)
	assert.Equal(t, cj[0].Name, "task1")
	assert.Equal(t, cj[0].Service.Name, "one")
	assert.Equal(t, cj[1].Service.Name, "two")
	assert.Equal(t, cj[1].Name, "task2")
	assert.Equal(t, cj[2].Service.Name, "two")
	assert.Equal(t, cj[2].Name, "task3")
}
