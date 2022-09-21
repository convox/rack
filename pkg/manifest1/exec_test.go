package manifest1

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunAsync(t *testing.T) {
	s := make(Stream)
	done := make(chan error)
	RunAsync(s, exec.Command("echo", "wow"), done, RunnerOptions{
		Verbose: false,
	})

	err := <-done
	require.NoError(t, err)
	require.Equal(t, "wow", <-s)
}
