package aws

import (
	"fmt"

	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) CertificateApply(app, service string, port int, id string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateCreate(pub, key string, opts structs.CertificateCreateOptions) (*structs.Certificate, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateDelete(id string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateGenerate(domains []string) (*structs.Certificate, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateList() (structs.Certificates, error) {
	return nil, fmt.Errorf("unimplemented")
}
