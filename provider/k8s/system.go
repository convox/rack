package k8s

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) SystemGet() (*structs.System, error) {
	s := &structs.System{
		Name:     p.Rack,
		Provider: p.Provider,
		Status:   "running",
		Version:  p.Version,
	}

	return s, nil
}

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	name := helpers.DefaultString(opts.Name, "convox")

	p.ID = helpers.DefaultString(opts.Id, "")
	p.Rack = name
	p.Socket = p.dockerSocket()

	if err := p.systemUpdate(helpers.DefaultString(opts.Version, "dev")); err != nil {
		return "", err
	}

	return "", nil
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go p.streamSystemLogs(w, opts)

	return r, nil
}

func (p *Provider) SystemMetrics(opts structs.MetricsOptions) (structs.Metrics, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	pds, err := p.Cluster.CoreV1().Pods(p.Rack).List(am.ListOptions{})
	if err != nil {
		return nil, err
	}

	pss := structs.Processes{}

	for _, pd := range pds.Items {
		ps, err := processFromPod(pd)
		if err != nil {
			return nil, err
		}

		ps.App = "system"

		pss = append(pss, *ps)
	}

	return pss, nil
}

func (p *Provider) SystemReleases() (structs.Releases, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUninstall(name string, w io.Writer, opts structs.SystemUninstallOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	if err := p.systemUpdate(helpers.DefaultString(opts.Version, p.Version)); err != nil {
		return err
	}

	return nil
}

func (p *Provider) streamSystemLogs(w io.WriteCloser, opts structs.LogsOptions) {
	defer w.Close()

	pl := func() (structs.Processes, error) {
		pss, err := p.SystemProcesses(structs.SystemProcessesOptions{})
		if err != nil {
			return nil, err
		}

		return pss, nil
	}

	ch := make(chan error)

	go p.streamProcessListLogs(w, pl, opts, ch)

	for {
		err, ok := <-ch
		if err != nil {
			fmt.Printf("err = %+v\n", err)
		}
		if !ok {
			return
		}
	}
}

func (p *Provider) systemUpdate(version string) error {
	log := p.logger.At("systemUpdate").Namespace("id=%s rack=%s version=%s", p.ID, p.Rack, version)

	params := map[string]interface{}{
		"Docker":  p.Socket,
		"ID":      p.ID,
		"Rack":    p.Rack,
		"Version": version,
	}

	if out, err := p.ApplyTemplate("custom", "system=convox,type=custom,scope=k8s", nil); err != nil {
		return log.Error(fmt.Errorf("update error: %s", string(out)))
	}

	if out, err := p.ApplyTemplate("metrics", "system=convox,type=metrics", nil); err != nil {
		return log.Error(fmt.Errorf("update error: %s", string(out)))
	}

	if out, err := p.ApplyTemplate("rack", fmt.Sprintf("system=convox,type=rack,rack=%s", p.Rack), params); err != nil {
		return log.Error(fmt.Errorf("update error: %s", string(out)))
	}

	return log.Success()
}
