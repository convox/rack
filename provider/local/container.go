package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type container struct {
	Command   []string
	Env       map[string]string
	Hostname  string
	Image     string
	Labels    map[string]string
	Listeners map[int]string
	Id        string
	Memory    int
	Name      string
	Port      int
	Volumes   []string
}

type containerPort struct {
	Container int
	Host      int
}

type containerTarget struct {
	FromScheme string
	FromPort   int
	ToScheme   string
	ToPort     int
}

type host struct {
	endpoints map[int]endpoint
}

type endpoint struct {
	protocol string
	targets  []string
}

// func (p *Provider) routeContainers(cc []container) error {
//   if p.router == nil {
//     return nil
//   }

//   if _, err := p.router.RackGet(p.Name); err != nil {
//     if err := p.routerRegister(); err != nil {
//       return err
//     }
//   }

//   hosts := map[string]host{}

//   for _, c := range cc {
//     if c.Targets == nil || len(c.Targets) == 0 {
//       continue
//     }

//     h := hosts[c.Hostname]

//     if h.endpoints == nil {
//       h.endpoints = map[int]endpoint{}
//     }

//     for _, t := range c.Targets {
//       e := h.endpoints[t.FromPort]

//       if e.targets == nil {
//         e.targets = []string{}
//       }

//       e.protocol = t.FromScheme

//       out, err := exec.Command("docker", "inspect", "-f", fmt.Sprintf(`{{index (index (index .NetworkSettings.Ports "%d/tcp") 0) "HostPort"}}`, t.ToPort), c.Name).CombinedOutput()
//       if err != nil {
//         return err
//       }

//       e.targets = append(e.targets, fmt.Sprintf("%s://127.0.0.1:%s", t.ToScheme, strings.TrimSpace(string(out))))

//       h.endpoints[t.FromPort] = e
//     }

//     hosts[c.Hostname] = h
//   }

//   for hostname, h := range hosts {
//     if _, err := p.router.HostGet(p.Name, hostname); err != nil {
//       if err := p.router.HostCreate(p.Name, hostname); err != nil {
//         return err
//       }
//     }

//     for port, e := range h.endpoints {
//       if _, err := p.router.EndpointGet(p.Name, hostname, port); err != nil {
//         if err := p.router.EndpointCreate(p.Name, hostname, e.protocol, port); err != nil {
//           return err
//         }
//       }

//       et, err := p.router.TargetList(p.Name, hostname, port)
//       if err != nil {
//         return err
//       }

//       for _, t := range missing(et, e.targets) {
//         if err := p.router.TargetRemove(p.Name, hostname, port, t); err != nil {
//           return err
//         }
//       }

//       for _, t := range missing(e.targets, et) {
//         if err := p.router.TargetAdd(p.Name, hostname, port, t); err != nil {
//           return err
//         }
//       }
//     }
//   }

//   return nil
// }

// func missing(master, check []string) []string {
//   r := []string{}

//   for _, m := range master {
//     found := false
//     for _, c := range check {
//       if m == c {
//         found = true
//         break
//       }
//     }
//     if !found {
//       r = append(r, m)
//     }
//   }

//   return r
// }

func (p *Provider) containerStart(c container, app, release string) (string, error) {
	if c.Name == "" {
		return "", fmt.Errorf("name required")
	}

	args := []string{"run", "--detach", "-t"}

	args = append(args, "--name", c.Name)

	for k, v := range c.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range c.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	if m := c.Memory; m > 0 {
		args = append(args, "--memory-reservation", fmt.Sprintf("%dm", m))
	}

	for _, v := range c.Volumes {
		args = append(args, "-v", v)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	args = append(args, "-e", fmt.Sprintf("APP=%s", app))
	args = append(args, "-e", fmt.Sprintf("RACK_URL=https://%s:3000", hostname))
	args = append(args, "-e", fmt.Sprintf("RELEASE=%s", release))
	args = append(args, "--link", hostname)

	if c.Port != 0 {
		args = append(args, "-p", strconv.Itoa(c.Port))
	}

	args = append(args, c.Image)
	args = append(args, c.Command...)

	exec.Command("docker", "rm", "-f", c.Name).Run()

	data, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", err
	}

	id := strings.TrimSpace(string(data))

	if len(id) < 12 {
		return "", fmt.Errorf("unable to start container")
	}

	return id[0:12], nil
}

func (p *Provider) containerStop(id string) error {
	return exec.Command("docker", "stop", "--time", "3", id).Run()
}

func (p *Provider) containerStopAsync(id string, wg *sync.WaitGroup) {
	defer wg.Done()
	p.containerStop(id)
}

func containerBinding(id string, bind string) (string, error) {
	data, err := exec.Command("docker", "inspect", "-f", "{{json .HostConfig.PortBindings}}", id).CombinedOutput()
	if err != nil {
		return "", err
	}

	var bindings map[string][]struct {
		HostPort string
	}

	if err := json.Unmarshal(data, &bindings); err != nil {
		return "", err
	}

	b, ok := bindings[bind]
	if !ok {
		return "", nil
	}
	if len(b) < 1 {
		return "", nil
	}

	return b[0].HostPort, nil
}

func containersByLabels(labels map[string]string) ([]container, error) {
	args := []string{}

	for k, v := range labels {
		args = append(args, "--filter", fmt.Sprintf("label=%s=%s", k, v))
	}

	return containerList(args...)
}

func containerIDs(args ...string) ([]string, error) {
	as := []string{"ps", "--format", "{{.ID}}"}

	as = append(as, args...)

	data, err := exec.Command("docker", as...).CombinedOutput()
	if err != nil {
		return nil, err
	}

	all := strings.TrimSpace(string(data))

	if all == "" {
		return []string{}, nil
	}

	return strings.Split(all, "\n"), nil
}

func containerList(args ...string) ([]container, error) {
	ids, err := containerIDs(args...)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []container{}, nil
	}

	as := []string{"inspect", "--format", "{{json .}}"}
	as = append(as, ids...)

	data, err := exec.Command("docker", as...).CombinedOutput()
	if err != nil {
		return nil, err
	}

	cs := []container{}

	d := json.NewDecoder(bytes.NewReader(data))

	for d.More() {
		var c struct {
			Id     string
			Name   string
			Config struct {
				Labels map[string]string
			}
			NetworkSettings struct {
				Ports map[string][]struct {
					HostIp   string
					HostPort string
				}
			}
		}

		if err := d.Decode(&c); err != nil {
			return nil, err
		}

		cc := container{
			Id:        c.Id,
			Labels:    c.Config.Labels,
			Listeners: map[int]string{},
			Name:      c.Name[1:],
			Hostname:  c.Config.Labels["convox.hostname"],
		}

		pi, err := strconv.Atoi(c.Config.Labels["convox.port"])
		if err != nil {
			return nil, err
		}

		cc.Port = pi

		for host, cp := range c.NetworkSettings.Ports {
			hpi, err := strconv.Atoi(strings.Split(host, "/")[0])
			if err != nil {
				return nil, err
			}

			if len(cp) == 1 {
				cc.Listeners[hpi] = fmt.Sprintf("127.0.0.1:%s", cp[0].HostPort)
			}
		}

		cs = append(cs, cc)
	}

	return cs, nil
}
