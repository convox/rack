package local

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func (p *Provider) Proxy(app, pid string, port int, in io.Reader) (io.ReadCloser, error) {
	log := p.logger("Proxy").Append("app=%q pid=%q port=%d", app, pid, port)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	data, err := exec.Command("docker", "inspect", pid, "--format", "{{.NetworkSettings.IPAddress}}").CombinedOutput()
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	ip := strings.TrimSpace(string(data))

	cn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	go io.Copy(cn, in)

	return cn, log.Success()
}
