package models_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aryann/difflib"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/manifest1"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("AWS_REGION", "test")
	os.Setenv("CLUSTER", "convox-test")
}

func TestFixtures(t *testing.T) {
	fixtures, err := availableFixtures()

	require.NotNil(t, fixtures)
	require.NoError(t, err)

	for _, fixture := range fixtures {
		assertFixture(t, fixture)
	}
}

func availableFixtures() ([]string, error) {
	fixtures := []string{}

	err := filepath.Walk("fixtures/", func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yml") {
			file := filepath.Base(path)
			name := file[0 : len(file)-4]
			fixtures = append(fixtures, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return fixtures, nil
}

func assertFixture(t *testing.T, name string) {
	orig := manifest1.ManifestRandomPorts
	manifest1.ManifestRandomPorts = false
	defer func() { manifest1.ManifestRandomPorts = orig }()

	app := models.App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}

	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.NoError(t, err)

	manifest, err := manifest1.Load(data)
	require.NoError(t, err)

	formation, err := app.Formation(*manifest)
	require.NoError(t, err)

	pretty, err := models.PrettyJSON(formation)
	require.NoError(t, err)

	data, err = ioutil.ReadFile(fmt.Sprintf("fixtures/%s.json", name))
	require.NoError(t, err)

	diff1 := strings.Split(strings.TrimSpace(string(data)), "\n")
	diff2 := strings.Split(strings.TrimSpace(pretty), "\n")

	diff := difflib.Diff(diff1, diff2)
	diffs := []string{}

	// bigger than max
	prev := 1000000

	for l, d := range diff {
		switch d.Delta {
		case difflib.LeftOnly:
			if (l - prev) > 1 {
				diffs = append(diffs, "")
			}
			diffs = append(diffs, fmt.Sprintf("%04d - %s", l, d.Payload))
			prev = l
		case difflib.RightOnly:
			if (l - prev) > 1 {
				diffs = append(diffs, "")
			}
			diffs = append(diffs, fmt.Sprintf("%04d + %s", l, d.Payload))
			prev = l
		}
	}

	if len(diffs) > 0 {
		t.Errorf("Unexpected results for %s:\n%s", name, strings.Join(diffs, "\n"))
	}
}
