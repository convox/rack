package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/rack/sync"
)

type Run struct {
	App   string
	Dir   string
	Cache bool
	Sync  bool

	done      chan error
	manifest  Manifest
	output    Output
	processes []Process
	proxies   []Proxy
	syncs     []sync.Sync
}

// NewRun Default constructor method for a Run object
func NewRun(dir, app string, m Manifest, cache, sync bool) Run {
	return Run{
		App:      app,
		Dir:      dir,
		Cache:    cache,
		Sync:     sync,
		manifest: m,
		output:   NewOutput(),
	}
}

func (r *Run) Start() error {
	if r.done != nil {
		return fmt.Errorf("already started")
	}

	if denv := filepath.Join(r.Dir, ".env"); exists(denv) {
		data, err := ioutil.ReadFile(denv)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))

		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "=") {
				parts := strings.SplitN(scanner.Text(), "=", 2)

				err := os.Setenv(parts[0], parts[1])
				if err != nil {
					return err
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	// check for required env vars
	existing := map[string]bool{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			existing[parts[0]] = true
		}
	}

	for _, s := range r.manifest.Services {
		links := map[string]bool{}

		for _, l := range s.Links {
			key := fmt.Sprintf("%s_URL", strings.ToUpper(l))
			links[key] = true
		}

		missingEnv := []string{}
		for key, val := range s.Environment {
			eok := val != ""
			_, exok := existing[key]
			_, lok := links[key]
			if !eok && !exok && !lok {
				missingEnv = append(missingEnv, key)
			}
		}

		if len(missingEnv) > 0 {
			return fmt.Errorf("env expected: %s", strings.Join(missingEnv, ", "))
		}
	}

	// preload system-level stream names
	r.output.Stream("convox")
	r.output.Stream("build")

	// preload process stream names so padding is set correctly
	for _, s := range r.manifest.runOrder() {
		r.output.Stream(s.Name)
	}

	r.done = make(chan error)

	err := r.manifest.Build(r.Dir, r.App, r.output.Stream("build"), r.Cache)
	if err != nil {
		return err
	}

	system := r.output.Stream("convox")

	for _, s := range r.manifest.runOrder() {
		proxies := s.Proxies(r.App)

		p := s.Process(r.App, r.manifest)

		Docker("rm", "-f", p.Name).Run()

		runAsync(r.output.Stream(p.service.Name), Docker(append([]string{"run"}, p.Args...)...), r.done)

		sp, err := p.service.SyncPaths()
		if err != nil {
			return err
		}

		if r.Sync {
			syncs := []sync.Sync{}

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
				go func(s sync.Sync) {
					s.Start(sync.Stream(system))
				}(s)
				r.syncs = append(r.syncs, s)
			}
		}

		r.processes = append(r.processes, p)

		waitForContainer(p.Name, s)

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
	args := []string{"stop"}

	for _, p := range r.proxies {
		args = append(args, p.Name)
	}

	for _, p := range r.processes {
		args = append(args, p.Name)
	}

	Docker(args...).Run()
}

func pruneSyncs(syncs []sync.Sync) []sync.Sync {
	pruned := []sync.Sync{}

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

func waitForContainer(container string, service Service) {
	i := 0

	for {
		host := containerHost(container, service.Networks)
		i += 1

		// wait 5s max
		if host != "" || i > 50 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}
