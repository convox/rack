package test

import (
	"bytes"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ExecRun struct {
	Command string
	Exit    int
	Stdout  string
	Stderr  string
}

func (er ExecRun) Test(t *testing.T) {
	stdout, stderr, code, err := Exec(er.Command)

	assert.Nil(t, err, "should be nil")
	assert.Equal(t, er.Exit, code, "exit code should be equal")
	assert.Equal(t, er.Stdout, stdout, "stdout should be equal")
	assert.Equal(t, er.Stderr, stderr, "stderr should be equal")
}

func Exec(command string) (string, string, int, error) {
	cmd := exec.Command("sh", "-c", command)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

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
