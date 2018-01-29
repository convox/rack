package local

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/pkg/errors"
)

func (p *Provider) FilesDelete(app, pid string, files []string) error {
	log := p.logger("FilesDelete").Append("app=%q pid=%q", app, pid)

	if _, err := p.AppGet(app); err != nil {
		return log.Error(err)
	}

	args := []string{"exec", pid, "rm", "-f"}
	args = append(args, files...)

	if err := exec.Command("docker", args...).Run(); err != nil {
		return errors.WithStack(log.Error(err))
	}

	return log.Success()
}

func (p *Provider) FilesUpload(app, pid string, r io.Reader) error {
	log := p.logger("FilesUpload").Append("app=%q pid=%q", app, pid)

	if _, err := p.AppGet(app); err != nil {
		return log.Error(err)
	}

	cmd := exec.Command("docker", "cp", "-", fmt.Sprintf("%s:.", pid))

	cmd.Stdin = r

	if err := cmd.Run(); err != nil {
		return errors.WithStack(log.Error(err))
	}

	return log.Success()
}
