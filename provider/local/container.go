package local

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type container struct {
	Command  []string
	Env      map[string]string
	Hostname string
	Image    string
	Labels   map[string]string
	Id       string
	Memory   int
	Name     string
	Targets  []containerTarget
	Volumes  []string
}

type containerPort struct {
	Container int
	Host      int
}

type containerTarget struct {
	Port   int
	Scheme string
	Target string
}

func (p *Provider) containerRegister(c container) error {
	if p.Router == "none" || c.Hostname == "" {
		return nil
	}

	// TODO: remove
	dt := http.DefaultTransport.(*http.Transport)
	dt.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	hc := http.Client{Transport: dt}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/endpoints/%s", p.Router, c.Hostname), nil)
	if err != nil {
		return err
	}

	res, err := hc.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	for _, t := range c.Targets {
		uv := url.Values{}

		uv.Add("scheme", t.Scheme)
		uv.Add("target", t.Target)

		req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/endpoints/%s/proxies/%d", p.Router, c.Hostname, t.Port), bytes.NewReader([]byte(uv.Encode())))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		res, err := hc.Do(req)
		if err != nil {
			return err
		}

		defer res.Body.Close()
	}

	return nil
}

func (p *Provider) containerStart(c container, app, release string) (string, error) {
	if c.Name == "" {
		return "", fmt.Errorf("name required")
	}

	args := []string{"run", "--detach"}

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
			Config struct {
				Labels map[string]string
			}
			HostConfig struct {
				PortBindings map[string][]struct {
					HostIp   string
					HostPort string
				}
			}
			Name string
		}

		if err := d.Decode(&c); err != nil {
			return nil, err
		}

		cc := container{
			Id:       c.Id,
			Labels:   c.Config.Labels,
			Name:     c.Name[1:],
			Hostname: c.Config.Labels["convox.hostname"],
		}

		app := c.Config.Labels["convox.app"]
		service := c.Config.Labels["convox.service"]
		scheme := c.Config.Labels["convox.scheme"]
		port := c.Config.Labels["convox.port"]

		if app != "" && service != "" && scheme != "" && port != "" {
			st := fmt.Sprintf("%s://rack/%s/service/%s:%s", scheme, app, service, port)

			cc.Targets = []containerTarget{
				containerTarget{Scheme: "http", Port: 80, Target: st},
				containerTarget{Scheme: "https", Port: 443, Target: st},
			}
		}

		cs = append(cs, cc)
	}

	return cs, nil
}
