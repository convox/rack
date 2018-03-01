// +build !windows

package local

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
	"github.com/kr/pty"
	"github.com/pkg/errors"
)

func (p *Provider) processExec(app, pid, command string, opts structs.ProcessExecOptions) (int, error) {
	log := p.logger("ProcessExec").Append("app=%q pid=%q command=%q", app, pid, command)

	if _, err := p.AppGet(app); err != nil {
		return 0, log.Error(err)
	}

	cmd := exec.Command("docker", "exec", "-it", pid, "sh", "-c", command)

	fd, err := pty.Start(cmd)
	if err != nil {
		return 0, errors.WithStack(log.Error(err))
	}

	go helpers.Pipe(fd, opts.Stream)

	if err := cmd.Wait(); err != nil {
		return 0, errors.WithStack(log.Error(err))
	}

	return 0, log.Success()
}

func (p *Provider) processRun(app string, opts structs.ProcessRunOptions) (string, error) {
	log := p.logger("ProcessRun").Append("app=%q", app)

	if opts.Name != nil {
		exec.Command("docker", "rm", "-f", *opts.Name).Run()
	}

	oargs, err := p.argsFromOpts(app, opts)
	if err != nil {
		return "", errors.WithStack(log.Error(err))
	}

	cmd := exec.Command("docker", oargs...)

	if opts.Stream != nil {
		rw, err := pty.Start(cmd)
		if err != nil {
			return "", errors.WithStack(log.Error(err))
		}

		go io.Copy(rw, opts.Stream)
		go io.Copy(opts.Stream, rw)
	} else {
		if err := cmd.Start(); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%d", cmd.Process.Pid), nil
}
