package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDockerRunning(t *testing.T) {
	cmd := exec.Command("docker", "ps")
	cmd.Stderr = os.Stderr
	cmd.Start()

	timer := time.AfterFunc(1*time.Second, func() {
		err := cmd.Process.Kill()
		if err != nil {
			panic(err) // panic as can't kill a process.
		}
	})
	err := cmd.Wait()
	timer.Stop()

	if err != nil {
		t.Errorf("Docker not running. try `boot2docker up`?")
	}
}

func TestDockerCompose(t *testing.T) {
	destDir, err := mkBuildDir("examples/docker-compose/")
	if err != nil {
		t.Errorf("ERROR %v", err)
	}
	defer os.RemoveAll(destDir)

	stdout, stderr := testBuild(destDir, "test")

	expect(t, grepManifest(stdout), `manifest|web:
manifest|  image: httpd
manifest|  ports:
manifest|    - 80:80`)

	expect(t, stderr, "")
}

func TestEnvFile(t *testing.T) {
	destDir, err := mkBuildDir("examples/env_file/")
	if err != nil {
		t.Errorf("ERROR %v", err)
	}
	defer os.RemoveAll(destDir)

	stdout, stderr := testBuild(destDir, "test")

	expect(t, grepManifest(stdout), `manifest|web:
manifest|  build: .
manifest|  env_file: .env
manifest|  environment: []
manifest|  ports: []`)

	expect(t, stderr, "")
}

func expect(t *testing.T, a interface{}, b interface{}) {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)

	if !bytes.Equal(aj, bj) {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func grepManifest(s string) string {
	lines := strings.Split(s, "\n")
	m := make([]string, 0)

	for i := range lines {
		if strings.HasPrefix(lines[i], "manifest|") {
			m = append(m, lines[i])
		}
	}

	return strings.Join(m, "\n")
}

func mkBuildDir(srcDir string) (string, error) {
	destDir, err := ioutil.TempDir("", "convox-build")

	if err != nil {
		return destDir, err
	}

	cpCmd := exec.Command("cp", "-rf", srcDir, destDir)
	err = cpCmd.Run()

	if err != nil {
		return destDir, err
	}

	return destDir, nil
}

func testBuild(repo, name string) (string, string) {
	// Capture stdout and stderr to strings via Pipes
	oldErr := os.Stderr
	oldOut := os.Stdout

	er, ew, _ := os.Pipe()
	or, ow, _ := os.Pipe()

	os.Stderr = ew
	os.Stdout = ow

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

	builder := NewBuilder()
	_ = builder.Build(repo, name, "", "", "", "")

	// restore stderr, stdout
	ew.Close()
	os.Stderr = oldErr
	err := <-errC

	ow.Close()
	os.Stdout = oldOut
	out := <-outC

	return out, err
}
