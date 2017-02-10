package manifest

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// RunnerOptions are optional settings for a Runner
type RunnerOptions struct {
	Verbose bool
}

// Runner is the interface to run commands
type Runner interface {
	CombinedOutput(cmd *exec.Cmd) ([]byte, error)
	Run(s Stream, cmd *exec.Cmd, opts RunnerOptions) error
	RunAsync(s Stream, cmd *exec.Cmd, dobe chan error, opts RunnerOptions)
}

type Exec struct{}

var DefaultRunner Runner = new(Exec)

//Run synchronously calls the command and pipes the output to the stream,
func (e *Exec) Run(s Stream, cmd *exec.Cmd, opts RunnerOptions) error {
	return run(s, cmd, opts)
}

//RunAsync synchronously calls the command and pipes the output to the stream,
func (e *Exec) RunAsync(s Stream, cmd *exec.Cmd, done chan error, opts RunnerOptions) {
	RunAsync(s, cmd, done, opts)
}

//CombinedOutput synchronously calls the command and returns the output,
//useful for internal checks
func (e *Exec) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

func run(s Stream, cmd *exec.Cmd, opts RunnerOptions) error {
	done := make(chan error, 1)
	RunAsync(s, cmd, done, opts)
	return <-done
}

// RunAsync runs a command asynchronously and streams the output
func RunAsync(s Stream, cmd *exec.Cmd, done chan error, opts RunnerOptions) {
	if opts.Verbose {
		s <- fmt.Sprintf("running: %s", strings.Join(cmd.Args, " "))
	}

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

	scanner.Buffer(make([]byte, 0, 4*1024), 10*1024*1024)

	for scanner.Scan() {
		s <- scanner.Text()
	}
}
