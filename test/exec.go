package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ExecRun struct {
	Command string
	Env     map[string]string
	Exit    int
	Stdin   string
	Stdout  string
	Stderr  string
}

func (er ExecRun) Test(t *testing.T) {
	stdout, stderr, code, err := er.exec()

	assert.Nil(t, err, "should be nil")
	assert.Equal(t, er.Exit, code, "exit code should be equal")
	assert.Equal(t, er.Stdout, stdout, "stdout should be equal")
	assert.Equal(t, er.Stderr, stderr, "stderr should be equal")
}

func (er ExecRun) exec() (string, string, int, error) {
	cmd := exec.Command("sh", "-c", er.Command)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.Env = os.Environ()

	for k, v := range er.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if er.Stdin != "" {
		cmd.Stdin = strings.NewReader(er.Stdin)
	}

	code := exitCode(cmd.Run())

	return stdout.String(), stderr.String(), code, nil
}

func exitCode(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		if status, ok := ee.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	if err != nil {
		return -1
	}

	return 0
}
