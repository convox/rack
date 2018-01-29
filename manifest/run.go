package manifest

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type RunOptions struct {
	Bind   bool
	Env    map[string]string
	Stdout io.Writer
	Stderr io.Writer
}

func (m *Manifest) Run(app string, opts RunOptions) error {
	ch := make(chan error)

	external := 0

	for _, s := range m.Services {
		if s.Port.Port > 0 {
			external++
		}
	}

	if external > 1 {
		return fmt.Errorf("can not currently start apps with more than one internet-facing service")
	}

	for _, s := range m.Services {
		go s.runAsync(ch, app, "", RunOptions{
			Bind:   opts.Bind,
			Env:    opts.Env,
			Stdout: m.Writer(s.Name, opts.Stdout),
			Stderr: m.Writer(s.Name, opts.Stderr),
		})

		if opts.Bind {
		}
	}

	for err := range ch {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s Service) run(ns string, command string, opts RunOptions) error {
	args := []string{"run"}

	container := fmt.Sprintf("%s-%s", ns, s.Name)

	for _, k := range strings.Split(s.EnvironmentKeys(), ",") {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, opts.Env[k]))
	}

	args = append(args, "--name", container)
	args = append(args, fmt.Sprintf("%s/%s", ns, s.Name))

	if command != "" {
		args = append(args, "sh", "-c", command)
	}

	go func() {
		if !opts.Bind {
			return
		}

		for {
			data, err := exec.Command("docker", "inspect", container, "--format", "{{.State.Running}}").CombinedOutput()
			if err != nil {
				fmt.Fprintf(opts.Stderr, "error: %s\n", err)
				return
			}

			if strings.TrimSpace(string(data)) == "true" {
				break
			}

			time.Sleep(1 * time.Second)
		}

		if err := createProxy(container, s.Port.Scheme, s.Port.Port); err != nil {
			fmt.Fprintf(opts.Stderr, "error: %s\n", err)
			return
		}
	}()

	exec.Command("docker", "rm", "-f", container).Run()

	if err := opts.docker(args...); err != nil {
		return err
	}

	return nil
}

func (s Service) runAsync(ch chan error, ns string, command string, opts RunOptions) {
	ch <- s.run(ns, command, opts)
}

func (o RunOptions) docker(args ...string) error {
	message(o.Stdout, "running: docker %s", strings.Join(args, " "))

	cmd := exec.Command("docker", args...)

	cmd.Stdout = o.Stdout
	cmd.Stderr = o.Stderr

	return cmd.Run()
}

func createProxy(container, scheme string, port int) error {
	secure := ""

	if scheme == "https" {
		secure = "secure"
	}

	if err := exec.Command("docker", "run", "-p", "80:3000", "--link", fmt.Sprintf("%s:host", container), "convox/proxy", "3000", fmt.Sprintf("%d", port), "http", secure).Start(); err != nil {
		return err
	}

	if err := exec.Command("docker", "run", "-p", "443:3000", "--link", fmt.Sprintf("%s:host", container), "convox/proxy", "3000", fmt.Sprintf("%d", port), "https", secure).Start(); err != nil {
		return err
	}

	return nil
}
