package manifest

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(s Stream, cmd *exec.Cmd) error
	RunAsync(s Stream, cmd *exec.Cmd, dobe chan error)
}

type Exec struct{}

var DefaultRunner Runner = new(Exec)

func (e *Exec) Run(s Stream, cmd *exec.Cmd) error {
	return run(s, cmd)
}

func (e *Exec) RunAsync(s Stream, cmd *exec.Cmd, done chan error) {
	runAsync(s, cmd, done)
}

func run(s Stream, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	runAsync(s, cmd, done)
	return <-done
}

func runAsync(s Stream, cmd *exec.Cmd, done chan error) {
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
