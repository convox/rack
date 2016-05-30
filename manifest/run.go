package manifest

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Run struct {
	App string
	Dir string

	done      chan error
	manifest  Manifest
	output    Output
	processes []Process
	proxies   []Proxy
	syncs     []Sync
}

func NewRun(dir, app string, m Manifest) Run {
	return Run{
		App:      app,
		Dir:      dir,
		manifest: m,
		output:   NewOutput(),
	}
}

func (r *Run) Start() error {
	if r.done != nil {
		return fmt.Errorf("already started")
	}

	// preload system-level stream names
	r.output.Stream("convox")
	r.output.Stream("build")

	// preload process stream names so padding is set correctly
	for _, s := range r.manifest.runOrder() {
		r.output.Stream(s.Name)
	}

	r.done = make(chan error)

	if err := r.manifest.Build(r.Dir, r.output.Stream("build")); err != nil {
		return err
	}

	system := r.output.Stream("convox")

	for _, s := range r.manifest.runOrder() {
		proxies := s.Proxies(r.App)

		p := s.Process(r.App)

		Docker("rm", "-f", p.Name).Run()

		runAsync(r.output.Stream(p.service.Name), Docker(append([]string{"run"}, p.Args...)...), r.done)

		sp, err := p.service.SyncPaths()

		if err != nil {
			return err
		}

		syncs := []Sync{}

		for local, remote := range sp {
			s, err := p.Sync(local, remote)

			if err != nil {
				return err
			}

			syncs = append(syncs, *s)
		}

		// remove redundant syncs
		syncs = pruneSyncs(syncs)

		for _, s := range syncs {
			go s.Start(system)
			r.syncs = append(r.syncs, s)
		}

		r.processes = append(r.processes, p)

		waitForContainer(p.Name)

		for _, proxy := range proxies {
			r.proxies = append(r.proxies, proxy)
			proxy.Start()
		}
	}

	return nil
}

func (r *Run) Wait() error {
	defer r.Stop()
	<-r.done
	return nil
}

func (r *Run) Stop() {
	for _, p := range r.processes {
		Docker("kill", p.Name).Run()
	}

	for _, p := range r.proxies {
		Docker("kill", p.Name).Run()
	}
}

func pruneSyncs(syncs []Sync) []Sync {
	pruned := []Sync{}

	for i := 0; i < len(syncs); i++ {
		root := true

		for j := 0; j < len(syncs); j++ {
			if i == j {
				continue
			}

			if syncs[j].Contains(syncs[i]) {
				root = false
				break
			}
		}

		if root {
			pruned = append(pruned, syncs[i])
		}
	}

	return pruned
}

func run(s Stream, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	runAsync(s, cmd, done)
	return <-done
}

func runAsync(s Stream, cmd *exec.Cmd, done chan error) {
	s <- fmt.Sprintf("running: %s", strings.Join(cmd.Args, " "))

	r, w := io.Pipe()

	go streamReader(s, r)

	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		done <- err
		return
	}

	go func() {
		done <- cmd.Wait()
	}()
}

func streamReader(s Stream, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		s <- scanner.Text()
	}
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
