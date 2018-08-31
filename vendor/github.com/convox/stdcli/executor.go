package stdcli

import (
	"io"
	"os"
	"os/exec"
)

type Executor interface {
	Execute(cmd string, args ...string) ([]byte, error)
	Run(w io.Writer, cmd string, args ...string) error
	Terminal(cmd string, args ...string) error
}

type CmdExecutor struct {
}

func (e *CmdExecutor) Execute(cmd string, args ...string) ([]byte, error) {
	return exec.Command(cmd, args...).CombinedOutput()
}

func (e *CmdExecutor) Run(w io.Writer, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)

	c.Stdout = w
	c.Stderr = w

	return c.Run()
}

func (e *CmdExecutor) Terminal(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)

	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}
