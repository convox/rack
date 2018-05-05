package local

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) processExec(app, pid, command string, opts structs.ProcessExecOptions) (int, error) {
	return -1, fmt.Errorf("unimplemented")
}

func (p *Provider) processRun(app, service string, opts structs.ProcessRunOptions) (string, error) {
	return "", fmt.Errorf("unimplemented")
}
