package manifest1

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerHostExposedPorts(t *testing.T) {
	tempDocker := Docker
	defer func() {
		Docker = tempDocker
	}()

	Docker = func(args ...string) *exec.Cmd {
		if args[0] == "ps" {
			return exec.Command("echo", "1\n2\n")
		}
		if args[0] == "inspect" {
			return exec.Command("echo", `{"3003/tcp":[{"HostIp":"0.0.0.0","HostPort":"3000"}]}`)
		}
		return exec.Command("echo", args...)
	}

	ports, err := DockerHostExposedPorts()
	require.NoError(t, err)
	require.Contains(t, ports, 3000)
}
