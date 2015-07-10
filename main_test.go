package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	yaml "github.com/convox/app/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Cases []struct {
	got, want interface{}
}

func TestManifestEntryNames(t *testing.T) {
	var manifest Manifest
	man := readFile(t, "fixtures", "web_postgis.yml")
	yaml.Unmarshal(man, &manifest)

	cases := Cases{
		{manifest.EntryNames(), []string{"postgres", "web"}},
	}

	_assert(t, cases)
}

func TestStagingWebPostgis(t *testing.T) {
	var manifest Manifest

	man := readFile(t, "fixtures", "web_postgis.yml")
	tmpl := readFile(t, "fixtures", "web_postgis.json")

	yaml.Unmarshal(man, &manifest)

	data, _ := buildTemplate("staging", "formation", func() string { return "12345" }, manifest)

	cases := Cases{
		{data, string(tmpl)},
	}

	_assert(t, cases)
}

func readFile(t *testing.T, dir string, name string) []byte {
	filename := filepath.Join(dir, name)

	dat, err := ioutil.ReadFile(filename)

	if err != nil {
		t.Errorf("ERROR readFile %v %v", filename, err)
	}

	return dat
}

func _assert(t *testing.T, cases Cases) {
	for _, c := range cases {
		j1, err := json.Marshal(c.got)

		if err != nil {
			t.Errorf("Marshal %q, error %q", c.got, err)
		}

		j2, err := json.Marshal(c.want)

		if err != nil {
			t.Errorf("Marshal %q, error %q", c.want, err)
		}

		if !bytes.Equal(j1, j2) {
			t.Errorf("Got %q, want %q", c.got, c.want)
		}
	}
}
