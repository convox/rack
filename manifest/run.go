package manifest

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/kr/text"
)

type RunOptions struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (s Service) run(ns string, command string, opts RunOptions) error {
	args := []string{"run", "-t"}

	args = append(args, fmt.Sprintf("%s/%s", ns, s.Name), "sh", "-c", command)

	return opts.docker(args...)
}

func (o RunOptions) docker(args ...string) error {
	message(o.Stdout, "running: docker %s", strings.Join(args, " "))

	cmd := exec.Command("docker", args...)

	cmd.Stdout = text.NewIndentWriter(o.Stdout, []byte("  "))
	cmd.Stderr = text.NewIndentWriter(o.Stderr, []byte("  "))

	return cmd.Run()
}
