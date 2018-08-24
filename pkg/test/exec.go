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
	Command    string
	Env        map[string]string
	Exit       int
	Dir        string
	Stdin      string
	Stdout     string
	OutMatch   string
	OutMatches []string
	Stderr     string
	Dump       bool
}

func (er ExecRun) Test(t *testing.T) {
	stdout, stderr, code, err := er.exec()

	if er.Dump {
		t.Log("ExecRun", er.Command, stdout, stderr, code, err)
	}

	assert.NoError(t, err)
	assert.Equal(t, er.Exit, code, fmt.Sprintf("exit code should be equal 🢂  %s", er.Command))
	if er.Stdout != "" {
		assert.Equal(t, er.Stdout, stdout, fmt.Sprintf("stdout should be equal 🢂  %s", er.Command))
	}
	if er.OutMatch != "" {
		assert.Contains(t, stdout, er.OutMatch,
			fmt.Sprintf("stdout %q should contain %q\n 🢂  %s", stdout, er.OutMatch, er.Command))
	}
	if er.Stderr != "" {
		assert.Contains(t, stderr, er.Stderr,
			fmt.Sprintf("stderr %q should contain %q\n 🢂  %s", stderr, er.Stderr, er.Command))
	}
}

func (er ExecRun) exec() (string, string, int, error) {
	cmd := exec.Command("sh", "-c", er.Command)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = er.Dir
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
