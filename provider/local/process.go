package local

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/pkg/errors"
)

const (
	ScannerStartSize = 4096
	ScannerMaxSize   = 1024 * 1024
)

func (p *Provider) ProcessExec(app, pid, command string, rw io.ReadWriter, opts structs.ProcessExecOptions) (int, error) {
	return p.processExec(app, pid, command, rw, opts)
}

func (p *Provider) ProcessGet(app, pid string) (*structs.Process, error) {
	if _, err := p.AppGet(app); err != nil {
		return nil, err
	}

	if strings.TrimSpace(pid) == "" {
		return nil, fmt.Errorf("pid cannot be blank")
	}

	data, err := exec.Command("docker", "inspect", pid, "--format", "{{.ID}}").CombinedOutput()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fpid := strings.TrimSpace(string(data))

	filters := []string{
		fmt.Sprintf("label=convox.app=%s", app),
		fmt.Sprintf("label=convox.rack=%s", p.Rack),
		fmt.Sprintf("id=%s", fpid),
	}

	pss, err := processList(filters, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(pss) != 1 {
		return nil, fmt.Errorf("no such process: %s", pid)
	}

	return &pss[0], nil
}

func (p *Provider) ProcessList(app string, opts structs.ProcessListOptions) (structs.Processes, error) {
	if _, err := p.AppGet(app); err != nil {
		return nil, err
	}

	filters := []string{
		fmt.Sprintf("label=convox.app=%s", app),
		fmt.Sprintf("label=convox.rack=%s", p.Rack),
	}

	if opts.Service != nil {
		filters = append(filters, fmt.Sprintf("label=convox.type=service"))
		filters = append(filters, fmt.Sprintf("label=convox.service=%s", *opts.Service))
	}

	pss, err := processList(filters, false)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return pss, nil
}

func (p *Provider) ProcessLogs(app, pid string, opts structs.LogsOptions) (io.ReadCloser, error) {
	log := p.logger("ProcessLogs").Append("app=%q pid=%q", app, pid)

	_, err := p.AppGet(app)
	if err != nil {
		return nil, log.Error(err)
	}

	ps, err := p.ProcessGet(app, pid)
	if err != nil {
		return nil, log.Error(err)
	}

	cr, cw := io.Pipe()

	args := []string{"logs"}

	if opts.Follow == nil || *opts.Follow {
		args = append(args, "-f")
	}

	if opts.Since != nil {
		args = append(args, "--since", time.Now().UTC().Add((*opts.Since)*-1).Format(time.RFC3339))
	}

	args = append(args, pid)

	cmd := exec.Command("docker", args...)

	cmd.Stdout = cw
	cmd.Stderr = cw

	if err := cmd.Start(); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	s := bufio.NewScanner(cr)

	s.Buffer(make([]byte, ScannerStartSize), ScannerMaxSize)

	rr, rw := io.Pipe()

	go func() {
		defer rw.Close()
		for s.Scan() {
			if opts.Prefix != nil && *opts.Prefix {
				fmt.Fprintf(rw, "%s %s/%s/%s %s\n", time.Now().Format(helpers.PrintableTime), ps.App, ps.Name, ps.Id, s.Text())
			} else {
				fmt.Fprintf(rw, "%s\n", s.Text())
			}
		}
		if err := s.Err(); err != nil {
			if opts.Prefix != nil && *opts.Prefix {
				fmt.Fprintf(rw, "%s %s/%s/%s scan error: %s\n", time.Now().Format(helpers.PrintableTime), ps.App, ps.Name, ps.Id, err)
			} else {
				fmt.Fprintf(rw, "scan error: %s\n", err)
			}
		}

	}()

	go func() {
		defer cw.Close()
		cmd.Wait()
	}()

	return rr, log.Success()
}

func (p *Provider) ProcessProxy(app, pid string, port int, in io.Reader) (io.ReadCloser, error) {
	_, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	data, err := exec.Command("docker", "inspect", pid, "--format", "{{.NetworkSettings.IPAddress}}").CombinedOutput()
	if err != nil {
		return nil, err
	}

	ip := strings.TrimSpace(string(data))

	cn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}

	go io.Copy(cn, in)

	return cn, nil
}

func (p *Provider) ProcessRun(app, service string, opts structs.ProcessRunOptions) (*structs.Process, error) {
	ropts := processStartOptions{
		Command:     cs(opts.Command, ""),
		Environment: opts.Environment,
		Memory:      ci(opts.Memory, 0),
		Release:     cs(opts.Release, ""),
	}

	pid, err := p.processRun(app, service, ropts)
	if err != nil {
		return nil, err
	}

	return p.ProcessGet(app, pid)
}

func (p *Provider) ProcessStop(app, pid string) error {
	log := p.logger("ProcessStop").Append("app=%q pid=%q", app, pid)

	if err := exec.Command("docker", "stop", "-t", "2", pid).Run(); err != nil {
		return errors.WithStack(log.Error(err))
	}

	return log.Success()
}

func (p *Provider) ProcessWait(app, pid string) (int, error) {
	log := p.logger("ProcessWait").Append("app=%q pid=%q", app, pid)

	pidi, err := strconv.Atoi(pid)
	if err != nil {
		return 0, err
	}

	ps, err := os.FindProcess(pidi)
	if err != nil {
		return 0, err
	}

	status, err := ps.Wait()
	if err != nil {
		return 0, err
	}

	if ws, ok := status.Sys().(syscall.WaitStatus); ok {
		return ws.ExitStatus(), log.Success()
	}

	return 0, log.Success()
}

type processStartOptions struct {
	Command     string
	Cpu         int
	Environment map[string]string
	Image       string
	Links       []string
	Memory      int
	Name        string
	Ports       map[string]string
	Release     string
	Volumes     map[string]string
}

func (p *Provider) argsFromOpts(app, service string, opts processStartOptions) ([]string, error) {
	args := []string{"run", "--rm", "-it", "-d"}

	release := opts.Release

	if release == "" {
		a, err := p.AppGet(app)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		release = a.Release
	}

	image := opts.Image

	if image == "" {
		m, r, err := helpers.ReleaseManifest(p, app, release)
		if err != nil {
			return nil, err
		}

		image = fmt.Sprintf("%s/%s:%s.%s", p.Rack, app, service, r.Build)

		s, err := m.Service(service)
		if err != nil {
			return nil, err
		}

		if s != nil {
			// manifest environment
			env, err := m.ServiceEnvironment(s.Name)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			for k, v := range env {
				args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
			}

			for _, sr := range s.Resources {
				for _, r := range m.Resources {
					if r.Name == sr {
						u, err := p.resourceURL(app, r.Type, r.Name)
						if err != nil {
							return nil, err
						}

						args = append(args, "-e", fmt.Sprintf("%s=%s", fmt.Sprintf("%s_URL", strings.ToUpper(sr)), u))
					}
				}
			}

			// app environment
			menv, err := helpers.AppEnvironment(p, app)
			if err != nil {
				return nil, err
			}

			for k, v := range menv {
				args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
			}

			// volumes
			s, err := m.Service(s.Name)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			vv, err := p.serviceVolumes(app, s.Volumes)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			for _, v := range vv {
				args = append(args, "-v", v)
			}
		}
	}

	// FIXME try letting docker daemon pass through dns
	// if this works long term can delete this
	// if p.Router != "" {
	//   args = append(args, "--dns", p.Router)
	// }

	if opts.Cpu != 0 {
		args = append(args, "--cpu-shares", fmt.Sprintf("%d", opts.Cpu))
	}

	for k, v := range opts.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, l := range opts.Links {
		args = append(args, "--link", l)
	}

	if opts.Memory != 0 {
		args = append(args, "--memory", fmt.Sprintf("%dM", opts.Memory))
	}

	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}

	for from, to := range opts.Ports {
		args = append(args, "-p", fmt.Sprintf("%s:%s", from, to))
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	args = append(args, "-e", fmt.Sprintf("APP=%s", app))
	args = append(args, "-e", fmt.Sprintf("RACK_URL=https://%s:5443", hostname))

	if opts.Release != "" {
		args = append(args, "-e", fmt.Sprintf("RELEASE=%s", opts.Release))
	}

	args = append(args, "--link", hostname)

	args = append(args, "--label", fmt.Sprintf("convox.app=%s", app))
	args = append(args, "--label", fmt.Sprintf("convox.rack=%s", p.Rack))
	args = append(args, "--label", fmt.Sprintf("convox.type=%s", "process"))
	args = append(args, "--label", fmt.Sprintf("convox.release=%s", release))
	args = append(args, "--label", fmt.Sprintf("convox.service=%s", service))

	for from, to := range opts.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", from, to))
	}

	args = append(args, image)

	if opts.Command != "" {
		args = append(args, "sh", "-c", opts.Command)
	}

	return args, nil
}

func processList(filters []string, all bool) (structs.Processes, error) {
	args := []string{"ps"}

	if all {
		args = append(args, "-a")
	}

	for _, f := range filters {
		args = append(args, "--filter", f)
	}

	args = append(args, "--format", "{{json .}}")

	data, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, err
	}

	pss := structs.Processes{}

	jd := json.NewDecoder(bytes.NewReader(data))

	for jd.More() {
		var dps struct {
			CreatedAt string
			Command   string
			ID        string
			Labels    string
			Ports     string
			Status    string
		}

		if err := jd.Decode(&dps); err != nil {
			return nil, err
		}

		labels := map[string]string{}

		for _, kv := range strings.Split(dps.Labels, ",") {
			parts := strings.SplitN(kv, "=", 2)

			if len(parts) == 2 {
				labels[parts[0]] = parts[1]
			}
		}

		if labels["convox.service"] == "" {
			continue
		}

		started, err := time.Parse("2006-01-02 15:04:05 -0700 MST", dps.CreatedAt)
		if err != nil {
			return nil, err
		}

		ps := structs.Process{
			Id:      dps.ID,
			App:     labels["convox.app"],
			Command: strings.Trim(dps.Command, `"`),
			Name:    labels["convox.service"],
			Release: labels["convox.release"],
			Started: started,
			Status:  "running",
			Ports:   []string{},
		}

		if parts := strings.Split(dps.Ports, "-\u003e"); len(parts) == 2 {
			host := strings.Split(parts[0], ":")[1]
			container := strings.Split(parts[1], "/")[0]
			ps.Ports = append(ps.Ports, fmt.Sprintf("%s:%s", host, container))
		}

		pss = append(pss, ps)
	}

	return pss, nil
}

// func (p *Provider) processStart(app, service, command string, opts processStartOptions) (string, error) {
//   log := p.logger("processStart").Append("app=%q service=%q command=%q", app, service, command)

//   if opts.Name != "" {
//     exec.Command("docker", "rm", "-f", opts.Name).Run()
//   }

//   if opts.Name == "" {
//     rs, err := helpers.RandomString(6)
//     if err != nil {
//       return "", errors.WithStack(log.Error(err))
//     }

//     opts.Name = fmt.Sprintf("%s.%s.process.%s.%s", p.Rack, app, service, rs)
//   }

//   oargs, err := p.argsFromOpts(app, service, command, opts)
//   if err != nil {
//     return "", errors.WithStack(log.Error(err))
//   }

//   args := append(oargs[0:1], "--detach")
//   args = append(args, oargs[1:]...)

//   data, err := exec.Command("docker", args...).CombinedOutput()
//   if err != nil {
//     return "", errors.WithStack(log.Error(err))
//   }

//   return strings.TrimSpace(string(data)), log.Success()
// }
