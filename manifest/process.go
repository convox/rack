package manifest

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/convox/rack/sync"
	"github.com/mitchellh/go-homedir"
)

type Process struct {
	Args []string
	Name string

	app      string
	manifest Manifest
	service  Service
}

type ArgOptions struct {
	Command     string
	IgnorePorts bool
	Name        string
}

func NewProcess(app string, s Service, m Manifest) Process {

	p := Process{
		Name:     fmt.Sprintf("%s-%s", app, s.Name),
		app:      app,
		manifest: m,
		service:  s,
	}

	p.Args = p.GenerateArgs(nil)

	return p
}

// GenerateArgs generates the argument list based on a process property
// Possible to optionally override certain fields via opts
func (p *Process) GenerateArgs(opts *ArgOptions) []string {
	args := []string{}

	args = append(args, "-i")
	args = append(args, "--rm")

	if opts == nil {
		opts = &ArgOptions{}
	}

	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	} else {
		args = append(args, "--name", p.Name)
	}

	if p.service.Entrypoint != "" {
		args = append(args, "--entrypoint", p.service.Entrypoint)
	}

	userEnv := map[string]string{}

	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)

		if len(parts) == 2 {
			userEnv[parts[0]] = parts[1]
		}
	}

	for k, v := range p.service.Environment {
		if v == "" {
			args = append(args, "-e", fmt.Sprintf("%s", k))
		} else {
			if ev, ok := userEnv[k]; ok {
				args = append(args, "-e", fmt.Sprintf("%s=%s", k, ev))
			} else {
				args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	for _, v := range p.service.ExtraHosts {
		args = append(args, "--add-host", v)
	}

	for _, n := range p.service.Networks {
		for _, in := range n {
			args = append(args, "--net", in.Name)
		}
	}

	for _, link := range p.service.Links {
		args = append(args, linkArgs(p.manifest.Services[link], fmt.Sprintf("%s-%s", p.app, link))...)
	}

	if !opts.IgnorePorts {
		for _, port := range p.service.Ports {
			args = append(args, "-p", port.String())
		}
	}

	for _, volume := range p.service.Volumes {
		if !strings.Contains(volume, ":") {
			home, err := homedir.Dir()
			if err != nil {
				log.Fatal(err)
			}
			hostPath, err := filepath.Abs(fmt.Sprintf("%s/.convox/volumes/%s/%s/%s", home, p.app, p.service.Name, volume))
			if err != nil {
				//this won't break
			}
			volume = fmt.Sprintf("%s:%s", hostPath, volume)
		}
		args = append(args, "-v", volume)
	}

	if p.service.Cpu != 0 {
		args = append(args, "--cpu-shares", strconv.FormatInt(p.service.Cpu, 10))
	}

	if p.service.Memory != 0 {
		args = append(args, "--memory", fmt.Sprintf("%#v", p.service.Memory))
	}

	args = append(args, p.service.Tag(p.app))

	if opts.Command != "" {
		args = append(args, "sh", "-c", opts.Command)
	} else if p.service.Command.String != "" {
		args = append(args, "sh", "-c", p.service.Command.String)
	} else if len(p.service.Command.Array) > 0 {
		args = append(args, p.service.Command.Array...)
	}

	return args
}

func (p *Process) Sync(local, remote string) (*sync.Sync, error) {
	return sync.NewSync(p.Name, local, remote)
}
