package models

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aryann/difflib"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/require"
)

func Diff(t *testing.T, name, s1, s2 string) {
	diff1 := strings.Split(strings.TrimSpace(s1), "\n")
	diff2 := strings.Split(strings.TrimSpace(s2), "\n")

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

type Cases []struct {
	got, want interface{}
}

func assertFixture(t *testing.T, name string, primary string) {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.Nil(t, err)

	app := &App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}
	manifest, err := LoadManifest(string(data), app)

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	formation, err := manifest.Formation()

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	data, err = ioutil.ReadFile(fmt.Sprintf("fixtures/%s.json", name))
	require.Nil(t, err)

	Diff(t, name, string(data), formation)
}

func assertFixtureUnbound(t *testing.T, name string, primary string) {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.Nil(t, err)

	app := &App{
		Name: "httpd",
		Tags: map[string]string{
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}
	manifest, err := LoadManifest(string(data), app)

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	for i, _ := range manifest {
		if manifest[i].Name == primary {
			manifest[i].primary = true
		}
	}

	formation, err := manifest.Formation()

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	data, err = ioutil.ReadFile(fmt.Sprintf("fixtures/%s_unbound.json", name))
	require.Nil(t, err)

	Diff(t, name, string(data), formation)
}

func TestManifestInvalid(t *testing.T) {
	manifest, err := LoadManifest("invalid-manifest", nil)

	assert.Nil(t, manifest)
	assert.NotNil(t, err)
	assert.Regexp(t, "^invalid manifest: ", err.Error())
}

func TestManifestFixtures(t *testing.T) {
	ManifestRandomPorts = false
	assertFixture(t, "multi_balancer", "web")
	assertFixture(t, "web_external_internal", "")
	assertFixture(t, "web_postgis", "")
	assertFixture(t, "web_postgis_internal", "")
	assertFixture(t, "worker", "")
	ManifestRandomPorts = true
}

// test unbound apps with old balancer names and primary process logic
func TestManifestFixtureUnbound(t *testing.T) {
	ManifestRandomPorts = false
	assertFixtureUnbound(t, "multi_balancer", "web")
	assertFixtureUnbound(t, "web_external_internal", "")
	assertFixtureUnbound(t, "web_postgis", "")
	assertFixtureUnbound(t, "web_postgis_internal", "")
	assertFixtureUnbound(t, "worker", "")
	ManifestRandomPorts = true
}

func TestManifestRandomPorts(t *testing.T) {
	manifest, err := LoadManifest("web:\n  ports:\n  - 80:3000\n  - 3001", nil)

	require.Nil(t, err)

	// kinda hacky but just making sure we're not in sequence here
	assert.NotEqual(t, 1, (manifest[0].randoms["3001"] - manifest[0].randoms["80:3000"]))
}
