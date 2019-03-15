package base

import (
	"fmt"
	"io"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) SystemGet() (*structs.System, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	return "", fmt.Errorf("unimplemented")
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemMetrics(opts structs.MetricsOptions) (structs.Metrics, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemReleases() (structs.Releases, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUninstall(name string, w io.Writer, opts structs.SystemUninstallOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
