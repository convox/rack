package exec

import (
	"io"
	"os"
	"os/exec"
)

type Exec struct {
}

func (e *Exec) Execute(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).CombinedOutput()
}

func (e *Exec) Run(w io.Writer, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func (e *Exec) Stream(w io.Writer, r io.Reader, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func (e *Exec) Terminal(command string, args ...string) error {
	return e.Stream(os.Stdout, os.Stdin, command, args...)
}
