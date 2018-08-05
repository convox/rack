package aws

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
)

func (p *Provider) SystemGet() (*structs.System, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemInstall(opts structs.SystemInstallOptions) (string, error) {
	return "", fmt.Errorf("unimplemented")
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.Reader, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemReleases() (structs.Releases, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUninstall(name string, opts structs.SystemUninstallOptions) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	return fmt.Errorf("unimplemented")
}
