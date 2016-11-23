package manifest

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Runner interface {
	CombinedOutput(cmd *exec.Cmd) ([]byte, error)
	Run(s Stream, cmd *exec.Cmd) error
	RunAsync(s Stream, cmd *exec.Cmd, dobe chan error)
}

type Exec struct{}

var DefaultRunner Runner = new(Exec)

//Run synchronously calls the command and pipes the output to the stream,
func (e *Exec) Run(s Stream, cmd *exec.Cmd) error {
	return run(s, cmd)
}

//RunAsync synchronously calls the command and pipes the output to the stream,
func (e *Exec) RunAsync(s Stream, cmd *exec.Cmd, done chan error) {
	RunAsync(s, cmd, done)
}

//CombinedOutput synchronously calls the command and returns the output,
//useful for internal checks
func (e *Exec) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

func run(s Stream, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	RunAsync(s, cmd, done)
	return <-done
}

func RunAsync(s Stream, cmd *exec.Cmd, done chan error) {
	s <- fmt.Sprintf("running: %s", strings.Join(cmd.Args, " "))

	r, w := io.Pipe()

	go streamReader(s, r)

	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		done <- err
		return
	}

	go func() {
		done <- cmd.Wait()
	}()
}

func streamReader(s Stream, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		s <- scanner.Text()
	}
}
