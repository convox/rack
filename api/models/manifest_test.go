package models

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aryann/difflib"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/stretchr/testify/require"
)

type Cases []struct {
	got, want interface{}
}

func assertFixture(t *testing.T, name string) {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.Nil(t, err)

	manifest, err := LoadManifest(string(data))
	require.Nil(t, err)

	formation, err := manifest.Formation()
	require.Nil(t, err)

	data, err = ioutil.ReadFile(fmt.Sprintf("fixtures/%s.json", name))
	require.Nil(t, err)

	diff1 := strings.Split(strings.TrimSpace(string(data)), "\n")
	diff2 := strings.Split(strings.TrimSpace(formation), "\n")

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

func TestManifestInvalid(t *testing.T) {
	manifest, err := LoadManifest("invalid-manifest")

	assert.Nil(t, manifest)
	assert.NotNil(t, err)
	assert.Regexp(t, "^invalid manifest: ", err.Error())
}

func TestManifestFixtures(t *testing.T) {
	ManifestRandomPorts = false
	assertFixture(t, "web_external_internal")
	assertFixture(t, "web_postgis")
	assertFixture(t, "web_postgis_internal")
	assertFixture(t, "worker")
	ManifestRandomPorts = true
}

func TestManifestRandomPorts(t *testing.T) {
	manifest, err := LoadManifest("web:\n  ports:\n  - 80:3000\n  - 3001")

	require.Nil(t, err)

	// kinda hacky but just making sure we're not in sequence here
	assert.NotEqual(t, 1, (manifest[0].randoms["3001"] - manifest[0].randoms["80:3000"]))
}
