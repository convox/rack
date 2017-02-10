package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/sync"
)

type Run struct {
	App       string
	Dir       string
	Opts      RunOptions
	Output    Output
	Processes map[string]Process

	done     chan error
	manifest Manifest
	proxies  []Proxy
	syncs    []sync.Sync
}

type RunOptions struct {
	Service string
	Command []string
	Build   bool
	Cache   bool
	Quiet   bool
	Sync    bool
}

// NewRun Default constructor method for a Run object
func NewRun(m Manifest, dir, app string, opts RunOptions) Run {
	return Run{
		App:       app,
		Dir:       dir,
		Opts:      opts,
		Processes: make(map[string]Process),
		manifest:  m,
		Output:    NewOutput(opts.Quiet),
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

	services, err := r.manifest.runOrder(r.Opts.Service)
	if err != nil {
		return err
	}

	for _, s := range services {
		links := map[string]bool{}

		for _, l := range s.Links {
			key := fmt.Sprintf("%s_URL", strings.ToUpper(l))
			links[key] = true
		}

		missingEnv := []string{}
		for _, e := range s.Environment {
			if e.Needed && e.Value == "" {
				if _, ok := existing[e.Name]; !ok {
					missingEnv = append(missingEnv, e.Name)
				}
			}
		}

		sort.Strings(missingEnv)

		if len(missingEnv) > 0 {
			return fmt.Errorf("%s missing from .env file", strings.Join(missingEnv, ", "))
		}
	}

	// preload system-level stream names
	r.Output.Stream("convox")
	r.Output.Stream("build")

	// preload process stream names so padding is set correctly
	for _, s := range services {
		r.Output.Stream(s.Name)
	}

	r.done = make(chan error)

	if r.Opts.Build {
		env := map[string]string{}

		for _, e := range os.Environ() {
			pp := strings.SplitN(e, "=", 2)

			if len(pp) == 2 {
				env[pp[0]] = pp[1]
			}
		}

		err = r.manifest.Build(r.Dir, r.App, r.Output.Stream("build"), BuildOptions{
			Environment: env,
			Cache:       r.Opts.Cache,
			Service:     r.Opts.Service,
		})
		if err != nil {
			return err
		}
	}

	system := r.Output.Stream("convox")

	for _, s := range services {
		proxies := s.Proxies(r.App)

		if r.Opts.Command != nil && len(r.Opts.Command) > 0 && s.Name == r.Opts.Service {
			s.Command.String = ""
			s.Command.Array = []string{"sh", "-c", strings.Join(r.Opts.Command, " ")}
		}

		p := s.Process(r.App, r.manifest)

		Docker("rm", "-f", p.Name).Run()

		RunAsync(r.Output.Stream(p.service.Name), Docker(append([]string{"run"}, p.Args...)...), r.done, RunnerOptions{Verbose: true})

		sp, err := p.service.SyncPaths()
		if err != nil {
			return err
		}

		if r.Opts.Sync {
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

		r.Processes[p.Name] = p

		if err := waitForContainer(p.Name, s); err != nil {
			return err
		}

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

	for _, p := range r.Processes {
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

func waitForContainer(container string, service Service) error {
	i := 0

	for {
		host := containerHost(container, service.Networks)
		i += 1

		if host != "" {
			return nil
		}

		// wait 60s max
		if i > 600 {
			return fmt.Errorf("%s failed to start within 60 seconds", container)
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
