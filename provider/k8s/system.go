package k8s

import (
	"bytes"
	"fmt"
	"io"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	systemTemplates = []string{"custom", "metrics", "rack", "router"}
)

func (p *Provider) SystemGet() (*structs.System, error) {
	status, err := p.Engine.SystemStatus()
	if err != nil {
		return nil, err
	}

	ss, _, err := p.atom.Status(p.Rack, "system")
	if err != nil {
		return nil, err
	}

	switch status {
	case "running", "unknown":
		status = helpers.AtomStatus(ss)
	}

	s := &structs.System{
		Name:     p.Rack,
		Provider: p.Provider,
		Status:   status,
		Version:  p.Version,
	}

	return s, nil
}

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	version := helpers.DefaultString(opts.Version, "dev")

	if err := p.systemUpdate(version); err != nil {
		return "", err
	}

	return "", nil
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
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

		ps.App = "rack"
		ps.Release = p.Version

		pss = append(pss, *ps)
	}

	pds, err = p.Cluster.CoreV1().Pods("convox-system").List(am.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pd := range pds.Items {
		ps, err := processFromPod(pd)
		if err != nil {
			return nil, err
		}

		ps.App = "system"
		ps.Release = p.Version

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

func (p *Provider) SystemTemplate(version string) ([]byte, error) {
	params := map[string]interface{}{
		"Version": version,
	}

	ts := [][]byte{}

	for _, st := range systemTemplates {
		data, err := p.RenderTemplate(fmt.Sprintf("system/%s", st), params)
		if err != nil {
			return nil, err
		}

		ldata, err := ApplyLabels(data, "system=convox,provider=k8s")
		if err != nil {
			return nil, err
		}

		ts = append(ts, ldata)
	}

	return bytes.Join(ts, []byte("---\n")), nil
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	version := helpers.DefaultString(opts.Version, p.Version)

	if err := p.systemUpdate(version); err != nil {
		return err
	}

	return nil
}

func (p *Provider) systemUpdate(version string) error {
	return nil
}
