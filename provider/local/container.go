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

		// dont list build processes
		if c.Config.Labels["convox.service"] == "build" {
			continue
		}

		cc := container{
			Id:        c.Id,
			Labels:    map[string]string{},
			Listeners: map[int]string{},
			Name:      c.Name[1:],
			Hostname:  c.Config.Labels["convox.hostname"],
		}

		for k, v := range c.Config.Labels {
			if strings.HasPrefix(k, "convox.") {
				cc.Labels[k] = v
			}
		}

		if c.Config.Labels["convox.port"] != "" {
			pi, err := strconv.Atoi(c.Config.Labels["convox.port"])
			if err != nil {
				return nil, err
			}

			cc.Port = pi
		}

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
