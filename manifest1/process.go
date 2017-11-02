package manifest1

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
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

	for _, e := range p.service.Environment {
		if e.Needed && e.Value == "" {
			args = append(args, "-e", fmt.Sprintf("%s", e.Name))
		} else {
			if ev, ok := userEnv[e.Name]; ok {
				args = append(args, "-e", fmt.Sprintf("%s=%s", e.Name, ev))
			} else {
				args = append(args, "-e", fmt.Sprintf("%s=%s", e.Name, e.Value))
			}
		}
	}

	for _, v := range p.service.ExtraHosts {
		args = append(args, "--add-host", v)
	}

	if p.service.Privileged {
		args = append(args, "--privileged")
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
			home := ""

			switch runtime.GOOS {
			case "windows":
				home = "/home/convox" // prefix with container path to use Docker Volume
			default:
				d, err := homedir.Dir() // prefix with host path to use OS File Sharing
				if err != nil {
					log.Fatal(err)
				}
				home, err = filepath.Abs(d)
				if err != nil {
					log.Fatal(err)
				}
			}

			volume = fmt.Sprintf(
				"%s:%s",
				filepath.Clean(fmt.Sprintf("%s/.convox/volumes/%s/%s/%s", home, p.app, p.service.Name, volume)),
				volume,
			)
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
