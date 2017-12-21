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

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/kr/pty"
	"github.com/pkg/errors"
)

func (p *Provider) ProcessExec(app, pid, command string, opts structs.ProcessExecOptions) (int, error) {
	log := p.logger("ProcessExec").Append("app=%q pid=%q command=%q", app, pid, command)

	if _, err := p.AppGet(app); err != nil {
		return 0, log.Error(err)
	}

	cmd := exec.Command("docker", "exec", "-it", pid, "sh", "-c", command)

	fd, err := pty.Start(cmd)
	if err != nil {
		return 0, errors.WithStack(log.Error(err))
	}

	go helpers.Pipe(fd, opts.Stream)

	if err := cmd.Wait(); err != nil {
		return 0, errors.WithStack(log.Error(err))
	}

	return 0, log.Success()
}

func (p *Provider) ProcessGet(app, pid string) (*structs.Process, error) {
	log := p.logger("ProcessGet").Append("app=%q pid=%q", app, pid)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	if strings.TrimSpace(pid) == "" {
		return nil, fmt.Errorf("pid cannot be blank")
	}

	data, err := exec.Command("docker", "inspect", pid, "--format", "{{.ID}}").CombinedOutput()
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	fpid := strings.TrimSpace(string(data))

	filters := []string{
		fmt.Sprintf("label=convox.app=%s", app),
		fmt.Sprintf("label=convox.rack=%s", p.Name),
		fmt.Sprintf("id=%s", fpid),
	}

	pss, err := processList(filters, true)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	if len(pss) != 1 {
		return nil, log.Error(fmt.Errorf("no such process: %s", pid))
	}

	return &pss[0], log.Success()
}

func (p *Provider) ProcessList(app string, opts structs.ProcessListOptions) (structs.Processes, error) {
	log := p.logger("ProcessGet").Append("app=%q", app)

	if _, err := p.AppGet(app); err != nil {
		return nil, log.Error(err)
	}

	filters := []string{
		fmt.Sprintf("label=convox.app=%s", app),
		fmt.Sprintf("label=convox.rack=%s", p.Name),
	}

	if opts.Service != "" {
		filters = append(filters, fmt.Sprintf("label=convox.type=service"))
		filters = append(filters, fmt.Sprintf("label=convox.service=%s", opts.Service))
	}

	pss, err := processList(filters, false)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return pss, log.Success()
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

	if opts.Follow {
		args = append(args, "-f")
	}

	args = append(args, pid)

	cmd := exec.Command("docker", args...)

	cmd.Stdout = cw
	cmd.Stderr = cw

	if err := cmd.Start(); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	s := bufio.NewScanner(cr)

	rr, rw := io.Pipe()

	go func() {
		defer rw.Close()
		for s.Scan() {
			if opts.Prefix {
				fmt.Fprintf(rw, "%s %s/%s/%s %s\n", time.Now().Format(helpers.PrintableTime), ps.App, ps.Name, ps.Id, s.Text())
			} else {
				fmt.Fprintf(rw, "%s\n", s.Text())
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

func (p *Provider) ProcessRun(app string, opts structs.ProcessRunOptions) (string, error) {
	log := p.logger("ProcessRun").Append("app=%q", app)

	if opts.Name != nil {
		exec.Command("docker", "rm", "-f", *opts.Name).Run()
	}

	oargs, err := p.argsFromOpts(app, opts)
	if err != nil {
		return "", errors.WithStack(log.Error(err))
	}

	cmd := exec.Command("docker", oargs...)

	if opts.Input != nil {
		rw, err := pty.Start(cmd)
		if err != nil {
			return "", errors.WithStack(log.Error(err))
		}
		defer rw.Close()

		go io.Copy(rw, opts.Input)
		go io.Copy(opts.Output, rw)
	} else {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output

		if err := cmd.Start(); err != nil {
			return "", errors.WithStack(log.Error(err))
		}
	}

	if err := cmd.Start(); err != nil {
		return "", errors.WithStack(log.Error(err))
	}

	return fmt.Sprintf("%d", cmd.Process.Pid), nil

	return "", log.Success()
}

func (p *Provider) ProcessStart(app string, opts structs.ProcessRunOptions) (string, error) {
	log := p.logger("ProcessStart").Append("app=%q", app)

	if opts.Name != nil {
		exec.Command("docker", "rm", "-f", *opts.Name).Run()
	}

	if opts.Name == nil {
		rs, err := helpers.RandomString(6)
		if err != nil {
			return "", errors.WithStack(log.Error(err))
		}

		opts.Name = options.String(fmt.Sprintf("%s.%s.process.%s.%s", p.Name, app, *opts.Service, rs))
	}

	oargs, err := p.argsFromOpts(app, opts)
	if err != nil {
		return "", errors.WithStack(log.Error(err))
	}

	args := append(oargs[0:1], "--detach")
	args = append(args, oargs[1:]...)

	data, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", errors.WithStack(log.Error(err))
	}

	return strings.TrimSpace(string(data)), log.Success()
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

func (p *Provider) argsFromOpts(app string, opts structs.ProcessRunOptions) ([]string, error) {
	args := []string{"run", "--rm", "-i"}

	if opts.Input != nil {
		args = append(args, "-t")
	}

	// if no release specified, use current release
	if opts.Release == nil {
		a, err := p.AppGet(app)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		opts.Release = options.String(a.Release)
	}

	// get release and manifest for initial environment and volumes
	var m *manifest.Manifest
	var release *structs.Release
	var service *manifest.Service
	var err error

	if opts.Release != nil {
		m, release, err = helpers.ReleaseManifest(p, app, *opts.Release)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// if service is not defined in manifest, i.e. "build", carry on
		service, err = m.Service(*opts.Service)
		if err != nil && !strings.Contains(err.Error(), "no such service") {
			return nil, errors.WithStack(err)
		}
	}

	if service != nil {
		// manifest environment
		env, err := m.ServiceEnvironment(service.Name)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for k, v := range env {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}

		// TODO reimplement
		// for _, resource := range s.Resources {
		//   k := strings.ToUpper(fmt.Sprintf("%s_URL", r.Name))
		//   args = append(args, "-e", fmt.Sprintf("%s=%s", k, r.Url))
		// }

		// app environment
		menv, err := helpers.AppEnvironment(p, app)
		if err != nil {
			return nil, err
		}

		for k, v := range menv {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}

		// volumes
		s, err := m.Service(service.Name)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, v := range s.Volumes {
			args = append(args, "-v", v)
		}
	}

	image := ""

	if opts.Image != nil {
		image = *opts.Image
	} else {
		image = fmt.Sprintf("%s/%s/%s:%s", p.Name, app, *opts.Service, release.Build)
	}

	// FIXME try letting docker daemon pass through dns
	// if this works long term can delete this
	// if p.Router != "" {
	//   args = append(args, "--dns", p.Router)
	// }

	for k, v := range opts.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, l := range opts.Links {
		args = append(args, "--link", l)
	}

	if opts.Memory != nil {
		args = append(args, "--memory", fmt.Sprintf("%dM", *opts.Memory))
	}

	if opts.Name != nil {
		args = append(args, "--name", *opts.Name)
	}

	for from, to := range opts.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", from, to))
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	args = append(args, "-e", fmt.Sprintf("APP=%s", app))
	args = append(args, "-e", fmt.Sprintf("RACK_URL=https://%s:3000", hostname))
	args = append(args, "-e", fmt.Sprintf("RELEASE=%s", opts.Release))

	args = append(args, "--link", hostname)

	args = append(args, "--label", fmt.Sprintf("convox.app=%s", app))
	args = append(args, "--label", fmt.Sprintf("convox.rack=%s", p.Name))
	args = append(args, "--label", fmt.Sprintf("convox.release=%s", opts.Release))
	args = append(args, "--label", fmt.Sprintf("convox.service=%s", opts.Service))
	args = append(args, "--label", fmt.Sprintf("convox.type=%s", "process"))

	for from, to := range opts.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", from, to))
	}

	args = append(args, image)

	if opts.Command != nil {
		args = append(args, "sh", "-c", *opts.Command)
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

	ps := structs.Processes{}

	jd := json.NewDecoder(bytes.NewReader(data))

	for jd.More() {
		var dps struct {
			CreatedAt string
			Command   string
			ID        string
			Labels    string
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

		ps = append(ps, structs.Process{
			Id:      dps.ID,
			App:     labels["convox.app"],
			Command: strings.Trim(dps.Command, `"`),
			Name:    labels["convox.service"],
			Release: labels["convox.release"],
			Started: started,
		})
	}

	return ps, nil
}
