package manifest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type Cases []struct {
	got, want interface{}
}

func TestDockerCompose(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)

	destDir := mkBuildDir(t, "../examples/docker-compose/")
	defer os.RemoveAll(destDir)

	m, _ := Generate(destDir)

	cases := Cases{
		{readFile(t, destDir, "docker-compose.yml"), `web:
  build: .
  links:
    - postgres
  ports:
    - 5000:3000
  volumes:
    - .:/app
postgres:
  image: convox/postgres
`},
		{[]string{"postgres", "web"}, m.runOrder()},
	}

	_assert(t, cases)
}

func TestDockerfile(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)

	destDir := mkBuildDir(t, "../examples/dockerfile/")
	defer os.RemoveAll(destDir)

	m, _ := Generate(destDir)

	cases := Cases{
		{readFile(t, destDir, "docker-compose.yml"), `main:
  build: .
  ports:
  - 5000:3000
`},
		{[]string{"main"}, m.runOrder()},
	}

	_assert(t, cases)
}

func TestProcfile(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)

	destDir := mkBuildDir(t, "../examples/procfile/")
	defer os.RemoveAll(destDir)

	m, _ := Generate(destDir)

	cases := Cases{
		{readFile(t, destDir, "docker-compose.yml"), `web:
  build: .
  command: ruby web.rb
  ports:
  - 5000:3000
worker:
  build: .
  command: ruby worker.rb
  ports:
  - 5100:3000
`},
		{[]string{"web", "worker"}, m.runOrder()},
	}

	_assert(t, cases)
}

func mkBuildDir(t *testing.T, srcDir string) string {
	destDir, err := ioutil.TempDir("", "")

	if err != nil {
		t.Errorf("ERROR mkBuildDir %v %v", srcDir, err)
		return destDir
	}

	cpCmd := exec.Command("cp", "-rf", srcDir, destDir)
	err = cpCmd.Run()

	if err != nil {
		t.Errorf("ERROR mkBuildDir %v %v", srcDir, err)
		return destDir
	}

	return destDir
}

func readFile(t *testing.T, dir string, name string) string {
	filename := filepath.Join(dir, name)

	dat, err := ioutil.ReadFile(filename)

	if err != nil {
		t.Errorf("ERROR readFile %v %v", filename, err)
	}

	return string(dat)
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
