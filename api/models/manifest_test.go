package models

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/aryann/difflib"
	"github.com/stretchr/testify/require"
)

type Cases []struct {
	got, want interface{}
}

func manifestFromFixture(t *testing.T, name string) Manifest {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.Nil(t, err)

	manifest, err := LoadManifest(string(data))
	require.Nil(t, err)

	return manifest
}

func compareToFixture(t *testing.T, manifest Manifest, name string) {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.json", name))
	require.Nil(t, err)

	formation, err := manifest.Formation()
	require.Nil(t, err)

	assertWithDiff(t, strings.TrimSpace(string(data)), strings.TrimSpace(formation))
}

func assertWithDiff(t *testing.T, s1, s2 string) {
	diff := difflib.Diff(strings.Split(s1, "\n"), strings.Split(s2, "\n"))
	diffs := []string{}

	for l, d := range diff {
		switch d.Delta {
		case difflib.LeftOnly:
			diffs = append(diffs, fmt.Sprintf("%04d - %s", l, d.Payload))
		case difflib.RightOnly:
			diffs = append(diffs, fmt.Sprintf("%04d + %s", l, d.Payload))
		}
	}

	if len(diffs) > 0 {
		t.Errorf("Unexpected result:\n%s", strings.Join(diffs, "\n"))
	}
}

func testFixture(t *testing.T, name string) {
	compareToFixture(t, manifestFromFixture(t, name), name)
}

func TestManifestWebPostgis(t *testing.T) {
	testFixture(t, "web_postgis")
}

func TestManifestWebPostgisInternal(t *testing.T) {
	testFixture(t, "web_postgis_internal")
}

func TestManifestWorker(t *testing.T) {
	testFixture(t, "worker")
}
