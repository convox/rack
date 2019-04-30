package kaws

import (
	"io"
	"os/exec"
	"strings"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) BuildExport(app, id string, w io.Writer) error {
	if err := p.authAppRepository(app); err != nil {
		return err
	}

	return p.Provider.BuildExport(app, id, w)
}

func (p *Provider) BuildImport(app string, r io.Reader) (*structs.Build, error) {
	if err := p.authAppRepository(app); err != nil {
		return nil, err
	}

	return p.Provider.BuildImport(app, r)
}

func (p *Provider) authAppRepository(app string) error {
	repo, _, err := p.RepositoryHost(app)
	if err != nil {
		return err
	}

	user, pass, err := p.RepositoryAuth(app)
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "login", "-u", user, "--password-stdin", repo)

	cmd.Stdin = strings.NewReader(pass)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
