package main

import (
	"bytes"
	"encoding/json"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

type Cases []struct {
	got, want interface{}
}

func TestStaging(t *testing.T) {
	var manifest Manifest

	man := []byte(`web:
  build: .
  links:
    - postgres
  ports:
    - 5000:3000
  volumes:
    - .:/app
postgres:
  image: convox/postgres
`)

	err := yaml.Unmarshal(man, &manifest)

	if err != nil {
		t.Errorf("ERROR %v", err)
	}

	for i, e := range manifest {
		for _ = range e.Ports {
			e.Randoms = append(e.Randoms, "12345")
		}
		manifest[i] = e
	}

	data, err := buildTemplate("staging", "formation", manifest)

	if err != nil {
		t.Errorf("ERROR %v", err)
	}

	var template Template

	if err := json.Unmarshal([]byte(data), &template); err != nil {
		t.Errorf("ERROR %v", err)
	}

	cases := Cases{
		{template.AWSTemplateFormatVersion, "2010-09-09"},
		{template.Parameters["WebPort5000Balancer"]["Default"], "5000"},
		{template.Parameters["WebPort5000Host"]["Default"], "12345"},
		{template.Resources["TaskDefinition"].Type, "Custom::ECSTaskDefinition"},
		{template.Resources["TaskDefinition"].Properties["Environment"], map[string]string{"Ref": "Environment"}},
	}

	_assert(t, cases)
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
