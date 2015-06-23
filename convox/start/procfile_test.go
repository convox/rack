package start

import (
	"bytes"
	"encoding/json"
	"testing"
)

type Cases []struct {
	got, want interface{}
}

func TestParseProcfile(t *testing.T) {
	p, err := parseProcfile([]byte(`web: ruby web.rb`))

	if err != nil {
		t.Errorf("TestParseProcfile err %q", err)
	}

	cases := Cases{
		{p, map[string]string{"web": "ruby web.rb"}},
	}

	_assert(t, cases)
}

func TestComposeFromProcfile(t *testing.T) {
	p1, _ := parseProcfile([]byte(`web: ruby web.rb`))
	d1, _ := composeFromProcfile(p1)

	p2, _ := parseProcfile([]byte("web: ruby web.rb\nworker: ruby worker.rb"))
	d2, _ := composeFromProcfile(p2)

	cases := Cases{
		{d1, []byte(`web:
  build: .
  command: ruby web.rb
  environment: []
  ports:
  - 5000:3000
`)},
		{d2, []byte(`web:
  build: .
  command: ruby web.rb
  environment: []
  ports:
  - 5000:3000
worker:
  build: .
  command: ruby worker.rb
  environment: []
  ports: []
`)},
	}

	_assert(t, cases)
}

func TestGenDockerfile(t *testing.T) {
	p, _ := parseProcfile([]byte(`web: ruby web.rb`))
	f, err := genDockerfile(p)

	if err != nil {
		t.Errorf("TestGenDockerfile err %q", err)
	}

	cases := Cases{
		{f, []byte(`FROM convox/cedar`)},
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
