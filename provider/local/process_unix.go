// +build !windows

package local

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/kr/pty"
	"github.com/pkg/errors"
)

func (p *Provider) processExec(app, pid, command string, rw io.ReadWriter, opts structs.ProcessExecOptions) (int, error) {
	log := p.logger("ProcessExec").Append("app=%q pid=%q command=%q", app, pid, command)

	if _, err := p.AppGet(app); err != nil {
		return 0, log.Error(err)
	}

	args := []string{"exec", "-it"}

	if opts.Height != nil {
		args = append(args, "-e", fmt.Sprintf("ROWS=%d", *opts.Height))
	}

	if opts.Width != nil {
		args = append(args, "-e", fmt.Sprintf("COLUMNS=%d", *opts.Width))
	}

	args = append(args, pid, "sh", "-c", command)

	cmd := exec.Command("docker", args...)

	fd, err := pty.Start(cmd)
	if err != nil {
		return 0, errors.WithStack(log.Error(err))
	}

	go helpers.Pipe(fd, rw)

	if err := cmd.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
				return ws.ExitStatus(), nil
			}
		}
		return 0, errors.WithStack(log.Error(err))
	}

	return 0, log.Success()
}

func (p *Provider) processRun(app, service string, opts processStartOptions) (string, error) {
	log := p.logger("ProcessRun").Append("app=%q", app)

	if opts.Name != "" {
		exec.Command("docker", "rm", "-f", opts.Name).Run()
	}

	oargs, err := p.argsFromOpts(app, service, opts)
	if err != nil {
		return "", errors.WithStack(log.Error(err))
	}

	pid, err := exec.Command("docker", oargs...).CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(pid)), nil
}
