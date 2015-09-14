package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aryann/difflib"
	yaml "gopkg.in/yaml.v2"
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
	manifest := readManifest(t, "fixtures", "web_postgis.yml")
	template := readFile(t, "fixtures", "web_postgis.json")

	data, _ := buildTemplate("staging", "formation", func() string { return "12345" }, manifest)

	cases := Cases{
		{strings.TrimSpace(data), strings.TrimSpace(string(template))},
	}

	_assert(t, cases)
}

func TestStagingWorker(t *testing.T) {
	manifest := readManifest(t, "fixtures", "worker.yml")
	template := readFile(t, "fixtures", "worker.json")

	data, _ := buildTemplate("staging", "formation", func() string { return "12345" }, manifest)

	cases := Cases{
		{strings.TrimSpace(data), strings.TrimSpace(string(template))},
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

func readManifest(t *testing.T, dir string, name string) Manifest {
	man := readFile(t, dir, name)

	var manifest Manifest
	err := yaml.Unmarshal(man, &manifest)

	if err != nil {
		t.Errorf("ERROR readManifest %v %v", filepath.Join(dir, name), err)
	}

	return manifest
}

func _assert(t *testing.T, cases Cases) {
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			if s1, ok := c.got.(string); ok {
				if s2, ok := c.want.(string); ok {
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

					t.Errorf("Unexpected result:\n%s", strings.Join(diffs, "\n"))
					return
				}
			}

			t.Errorf("%q\n%q\n", c.got, c.want)
		}
	}
}
