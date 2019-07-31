package local

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

func (p *Provider) watchForProcessCompletion(ctx context.Context, app, pid string, cancel func()) {
	defer cancel()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if ps, err := p.ProcessGet(app, pid); err != nil || (ps != nil && ps.Status == "complete") {
				time.Sleep(2 * time.Second)
				cancel()
				return
			}
		}
	}
}

func dockerDesktopCommand(command string) *exec.Cmd {
	return exec.Command("docker", "run", "-i", "--privileged", "--rm", "--privileged", "--pid=host", "debian", "nsenter", "-t", "1", "-m", "-u", "-n", "-i", "sh", "-c", command)
}

func installDockerDesktopCertificate(data []byte) error {
	cmd := dockerDesktopCommand("cat >> /run/config/etc/ssl/certs/ca-certificates.crt")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Run()

	dockerDesktopCommand("killall -HUP dockerd").Run()

	exec.Command("killall", "Docker").Run()
	exec.Command("open", "-a", "Docker", "--background").Run()

	for {
		if err := checkKubectl(); err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}
