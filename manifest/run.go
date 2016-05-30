package manifest

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
)

type Run struct {
	App string
	Dir string

	manifest  Manifest
	processes []Process
	proxies   []Proxy
}

func NewRun(dir, app string, m Manifest) Run {
	return Run{
		App:      app,
		Dir:      dir,
		manifest: m,
	}
}

func (r *Run) Start() error {
	defer r.Stop()

	done := make(chan error)
	kill := make(chan os.Signal, 1)

	signal.Notify(kill, os.Interrupt, os.Kill)

	if err := r.manifest.Build(r.Dir); err != nil {
		return err
	}

	for _, s := range r.manifest.runOrder() {
		proxies := s.Proxies(r.App)

		p := s.Process(r.App)

		Docker("rm", "-f", p.Name).Run()

		if err := runPrefixAsync(manifestPrefix(r.manifest, p.service.Name), Docker(append([]string{"run"}, p.Args...)...), done); err != nil {
			return err
		}

		r.processes = append(r.processes, p)

		waitForContainer(p.Name)

		for _, proxy := range proxies {
			r.proxies = append(r.proxies, proxy)
			proxy.Start()
		}
	}

	waitFor(done, kill)

	r.Stop()

	return nil
}

func waitFor(done chan error, kill chan os.Signal) {
	for {
		select {
		case <-kill:
			fmt.Println()
			return
		case <-done:
			return
		}
	}
}

func (r *Run) Stop() {
	for _, p := range r.processes {
		Docker("kill", p.Name).Run()
	}
}

func runPrefix(prefix string, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	runPrefixAsync(prefix, cmd, done)
	return <-done
}

func runPrefixAsync(prefix string, cmd *exec.Cmd, done chan error) error {
	printWrap(prefix, fmt.Sprintf("running: %s", strings.Join(cmd.Args, " ")))

	r, w := io.Pipe()

	go prefixReader(prefix, r)

	cmd.Stdout = w
	cmd.Stderr = w

	err := cmd.Start()

	go func() {
		done <- cmd.Wait()
	}()

	return err
}

func waitForContainer(container string) {
	i := 0

	for {
		host := containerHost(container)
		i += 1

		// wait 5s max
		if host != "" || i > 50 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}
