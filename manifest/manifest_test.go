package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	yaml "github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type Cases []struct {
	got, want interface{}
}

func TestBuild(t *testing.T) {
	destDir := mkBuildDir(t, "../examples/docker-compose")
	defer os.RemoveAll(destDir)

	m, _ := Generate(destDir)

	stdout, stderr := manifestBuild(m, "docker-compose")

	cases := Cases{
		{stdout, `RUNNING: docker build -t xvlbzgbaic .
RUNNING: docker pull convox/postgres
RUNNING: docker tag -f convox/postgres docker-compose/postgres
RUNNING: docker tag -f xvlbzgbaic docker-compose/web
`},
		{stderr, ""},
	}

	_assert(t, cases)
}

func TestPortsWanted(t *testing.T) {
	destDir := mkBuildDir(t, "../examples/docker-compose")
	defer os.RemoveAll(destDir)

	m, _ := Generate(destDir)
	ps, _ := m.PortsWanted()

	cases := Cases{
		{ps, []int64{5000}},
	}

	_assert(t, cases)
}

func TestRunOrder(t *testing.T) {
	var m Manifest
	data := []byte(`web:
  links:
    - postgres
    - redis
worker_2:
  links:
    - postgres
    - redis
worker_1:
  links:
    - postgres
    - redis
redis:
  image: convox/redis
postgres:
  image: convox/postgres
`)

	_ = yaml.Unmarshal(data, &m)

	cases := Cases{
		{m.runOrder(), []string{"postgres", "redis", "web", "worker_1", "worker_2"}},
	}

	_assert(t, cases)
}

func TestRun(t *testing.T) {
	destDir := mkBuildDir(t, "../examples/docker-compose")
	defer os.RemoveAll(destDir)

	m, _ := Generate(destDir)

	stdout, stderr := manifestRun(m, "docker-compose")

	cases := Cases{
		{stdout, fmt.Sprintf("\x1b[36mpostgres |\x1b[0m running: docker run -i --name docker-compose-postgres --rm=true docker-compose/postgres\n\x1b[33mweb      |\x1b[0m running: docker run -i --name docker-compose-web --rm=true --link docker-compose-postgres:postgres -p 5000:3000 -v %s:/app docker-compose/web\n", destDir)},
		{stderr, ""},
	}

	_assert(t, cases)
}

func TestGenerateDockerCompose(t *testing.T) {
	destDir := mkBuildDir(t, "../examples/docker-compose")
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

func TestGenerateDockerfile(t *testing.T) {
	destDir := mkBuildDir(t, "../examples/dockerfile")
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

func TestGenerateProcfile(t *testing.T) {
	destDir := mkBuildDir(t, "../examples/procfile")
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

	cpCmd := exec.Command("rsync", "-av", srcDir+"/", destDir)
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

func manifestBuild(m *Manifest, app string) (string, string) {
	oldErr := os.Stderr
	oldOut := os.Stdout

	er, ew, _ := os.Pipe()
	or, ow, _ := os.Pipe()

	Stderr = ew
	Stdout = ow

	Execer = func(bin string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	SignalWaiter = func(c chan os.Signal) error {
		return nil
	}

	errC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, er)
		errC <- buf.String()
	}()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, or)
		outC <- buf.String()
	}()

	m.Build(app)

	// restore stderr, stdout
	ew.Close()
	os.Stderr = oldErr
	err := <-errC

	ow.Close()
	os.Stdout = oldOut
	out := <-outC

	return out, err
}

func manifestRun(m *Manifest, app string) (string, string) {
	oldErr := os.Stderr
	oldOut := os.Stdout

	er, ew, _ := os.Pipe()
	or, ow, _ := os.Pipe()

	os.Stderr = ew
	os.Stdout = ow

	Execer = func(bin string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	SignalWaiter = func(c chan os.Signal) error {
		return nil
	}

	errC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, er)
		errC <- buf.String()
	}()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, or)
		outC <- buf.String()
	}()

	m.Run(app)

	// restore stderr, stdout
	ew.Close()
	os.Stderr = oldErr
	err := <-errC

	ow.Close()
	os.Stdout = oldOut
	out := <-outC

	return out, err
}
