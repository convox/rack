package local

import (
	"fmt"

	"github.com/convox/rack/structs"
)

func (p *Provider) CertificateApply(app, service string, port int, id string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateCreate(pub, key, chain string) (*structs.Certificate, error) {
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
