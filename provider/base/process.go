package aws

import (
	"fmt"
	"io"

	"github.com/convox/rack/structs"
)

func (p *Provider) ProcessExec(app, pid, command string, rw io.ReadWriter, opts structs.ProcessExecOptions) (int, error) {
	return 0, fmt.Errorf("unimplemented")
}

func (p *Provider) ProcessGet(app, pid string) (*structs.Process, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ProcessList(app string, opts structs.ProcessListOptions) (structs.Processes, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ProcessLogs(app, pid string, opts structs.LogsOptions) (io.Reader, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ProcessRun(app, service string, opts structs.ProcessRunOptions) (*structs.Process, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) ProcessStop(app, pid string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) ProcessWait(app, pid string) (int, error) {
	return 0, fmt.Errorf("unimplemented")
}
