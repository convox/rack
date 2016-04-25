package manifest

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type Cases []struct {
	got, want interface{}
}

const defaultManifestFile = "docker-compose.yml"

func init() {
	// default color to off during tests
	color.NoColor = true
}

func TestBuild(t *testing.T) {
	t.Skip("skipping until i can figure out regex")
	return

	destDir := mkBuildDir(t, "../../examples/compose")
	defer os.RemoveAll(destDir)

	_, _ = Init(destDir)
	m, _ := Read(destDir, defaultManifestFile)

	stdout, stderr := testBuild(m, "compose")

	cases := Cases{
		{stdout, `RUNNING: docker build -t xvlbzgbaic .
RUNNING: docker pull convox/postgres
RUNNING: docker tag -f convox/postgres compose/postgres
RUNNING: docker tag -f xvlbzgbaic compose/web
`},
		{stderr, ""},
	}

	_assert(t, cases)
}

func TestPortsWanted(t *testing.T) {
	destDir := mkBuildDir(t, "../../examples/compose")
	defer os.RemoveAll(destDir)

	_, _ = Init(destDir)
	m, _ := Read(destDir, defaultManifestFile)
	ps := m.PortsWanted()

	cases := Cases{
		{ps, []string{"5000"}},
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

type TestCommand struct {
	Command string
	Args    []string
	Output  string
}

func TestRun(t *testing.T) {
	destDir := mkBuildDir(t, "../../examples/compose")
	defer os.RemoveAll(destDir)

	_, _ = Init(destDir)
	m, _ := Read(destDir, defaultManifestFile)

	commands := []TestCommand{
		TestCommand{Command: "docker", Args: []string{"inspect"},
			Output: `[{"Config":{"Env":["POSTGRES_USERNAME=foo"]}}]`},
		TestCommand{Command: "docker", Args: []string{"run", "convox/docker-gateway"},
			Output: `1.1.1.1`},
	}

	//NOTE: this is a compromise on top of another compromise
	Execer = func(bin string, args ...string) *exec.Cmd {
		found := false

		for _, c := range commands {
			if c.Command == bin {
				found = true
				for i, arg := range c.Args {
					found = found && args[i] == arg
				}
			}

			if found {
				return exec.Command("echo", c.Output)
			}
		}

		return exec.Command("true")
	}

	stdout, stderr := testRun(m, "compose")

	cases := Cases{
		{stdout, "\x1b[36mpostgres |\x1b[0m docker run -i --name compose-postgres -p 5432:5432 compose/postgres\n\x1b[33mweb      |\x1b[0m docker run -i --name compose-web -e POSTGRES_HOST=1.1.1.1 -e POSTGRES_PASSWORD= -e POSTGRES_PATH= -e POSTGRES_PORT=5432 -e POSTGRES_SCHEME=tcp -e POSTGRES_URL=tcp://1.1.1.1:5432 -e POSTGRES_USERNAME= -p 5000:3000 compose/web\n"},
		{stderr, ""},
	}

	_assert(t, cases)
}

func TestGenerateDockerCompose(t *testing.T) {
	destDir := mkBuildDir(t, "../../examples/compose")
	defer os.RemoveAll(destDir)

	_, _ = Init(destDir)
	m, _ := Read(destDir, defaultManifestFile)

	Execer = func(bin string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	cases := Cases{
		{readFile(t, destDir, "docker-compose.yml"), `web:
  build: .
  links:
    - postgres
  ports:
    - 5000:3000
postgres:
  image: convox/postgres
  ports:
    - 5432
`},
		{[]string{"postgres", "web"}, m.runOrder()},
	}

	_assert(t, cases)
}

func TestGenerateDockerfile(t *testing.T) {
	destDir := mkBuildDir(t, "../../examples/dockerfile")
	defer os.RemoveAll(destDir)

	_, _ = Init(destDir)
	m, _ := Read(destDir, defaultManifestFile)

	Execer = func(bin string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

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
	destDir := mkBuildDir(t, "../../examples/procfile")
	defer os.RemoveAll(destDir)

	_, _ = Init(destDir)
	m, _ := Read(destDir, defaultManifestFile)

	Execer = func(bin string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

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

func TestInvalidYaml(t *testing.T) {
	m, err := Read("testdata", "invalid-merge.yml")
	// TODO: this should actually be an error
	require.NoError(t, err, "Unexpected error reading YAML testdata")

	t.Log((*m)["web"].Environment)
	t.Log((*m)["worker"].Environment)
}

func TestArrayMergeYaml(t *testing.T) {
	m, err := Read("testdata", "array-merge.yml")
	require.NoError(t, err, "Unexpected error reading YAML testdata")

	expected := []interface{}{"FOO=bar", "BAZ=qux"}
	assert.Equal(t, expected, (*m)["main"].Environment)
	assert.Equal(t, expected, (*m)["worker"].Environment)
}

func TestMapMergeYaml(t *testing.T) {
	m, err := Read("testdata", "map-merge.yml")
	require.NoError(t, err, "Unexpected error reading YAML testdata")

	expected := map[interface{}]interface{}{"FOO": "bar", "BAZ": "qux"}
	assert.Equal(t, expected, (*m)["main"].Environment)
	assert.Equal(t, expected, (*m)["worker"].Environment)
}

func TestEnvMapToSlice(t *testing.T) {
	m, err := Read("testdata", "map.yml")
	require.NoError(t, err, "Unexpected error reading YAML testdata")

	expected := map[interface{}]interface{}{"FOO": "bar", "BAZ": "qux"}
	assert.Equal(t, expected, (*m)["main"].Environment)

	data, err := m.Raw()
	require.NoError(t, err)

	// models/manifest.go expects a slice of strings
	var entries map[string]struct {
		Env []string `yaml:"environment"`
	}
	err = yaml.Unmarshal(data, &entries)
	require.NoError(t, err, "Error reading exported YAML")

	env := (*m)["main"].Environment.([]string)
	sort.Strings(env)
	assert.EqualValues(t, []string{"BAZ=qux", "FOO=bar"}, env)
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
			t.Errorf("Data Mismatch\nGot:  %q\nWant: %q", c.got, c.want)
		}
	}
}

type runnerFn func()

func testBuild(m *Manifest, app string) (string, string) {
	return testRunner(m, app, func() { m.Build(app, ".", true) })
}

func testRun(m *Manifest, app string) (string, string) {
	return testRunner(m, app, func() { m.Run(app, true) })
}

func testRunner(m *Manifest, app string, fn runnerFn) (string, string) {
	oldErr := os.Stderr
	oldOut := os.Stdout

	er, ew, _ := os.Pipe()
	or, ow, _ := os.Pipe()

	os.Stderr = ew
	os.Stdout = ow

	Stderr = ew
	Stdout = ow

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

	fn()

	// restore stderr, stdout
	ew.Close()
	os.Stderr = oldErr
	err := <-errC

	ow.Close()
	os.Stdout = oldOut
	out := <-outC

	return out, err
}
