package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

/* convox/build integration tests

This file tests convox/build by invoking the image in docker with
different example apps (in examples/)

The `TestDockerRunning` test is invoked via make test to ensure
docker is avaible. Make also builds convox/build before the tests.

*/

func TestGitUrl(t *testing.T) {
	out := runBuild(t, "worker", "https://github.com/convox-examples/worker.git")
	expected := `manifest|worker:
manifest|  build: .`
	actual := grepPrefix("manifest", out)
	if actual != expected {
		t.Errorf("Expected:\n %s \n got: \n %s", expected, actual)
	}
}

func TestDockerCompose(t *testing.T) {
	out := runBuild(t, "test", "examples/docker-compose/")
	expected := `manifest|web:
manifest|  image: httpd
manifest|  ports:
manifest|  - 80:80`
	actual := grepPrefix("manifest", out)
	if actual != expected {
		t.Errorf("Expected:\n %s \n got: \n %s", expected, actual)
	}
}

func TestEnvFile(t *testing.T) {
	// cleanup generated docker-compose.yml
	defer os.Remove("examples/env_file/docker-compose.yml")
	out := runBuild(t, "test", "examples/env_file")
	expected := `manifest|main:
manifest|  build: .
manifest|  ports: []`
	actual := grepPrefix("manifest", out)
	if actual != expected {
		t.Errorf("Expected:\n %s \n got: \n %s", expected, actual)
	}
}

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

func grepPrefix(prefix, s string) string {
	lines := strings.Split(s, "\n")
	m := make([]string, 0)

	for i := range lines {
		if strings.HasPrefix(lines[i], prefix+"|") {
			m = append(m, lines[i])
		}
	}

	return strings.Join(m, "\n")
}

func runBuild(t *testing.T, name, source string) string {
	var cmd *exec.Cmd

	fi, err := os.Stat(source)
	if err != nil || !fi.IsDir() {
		cmd = exec.Command("docker", "run", "-v", "/var/run/docker.sock:/var/run/docker.sock",
			"convox/build", name, source)
	} else {
		// if source is a directory then mount it
		hostPath, _ := filepath.Abs(source)
		vmPath := "/convox-build"
		args := []string{"run", "-v", "/var/run/docker.sock:/var/run/docker.sock",
			"-v", hostPath + "/:" + vmPath, "convox/build", name, vmPath}
		cmd = exec.Command("docker", args...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	return string(out)
}
