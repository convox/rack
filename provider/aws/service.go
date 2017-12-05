package aws

import (
	"fmt"
	"strings"

	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) ServiceList(app string) (structs.Services, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	switch a.Tags["Generation"] {
	case "", "1":
		return p.serviceListGeneration1(a)
	case "2":
	default:
		return nil, fmt.Errorf("unknown generation for app: %s", app)
	}

	r, err := p.ReleaseGet(a.Name, a.Release)
	if err != nil {
		return nil, err
	}

	env, err := p.EnvironmentGet(app)
	if err != nil {
		return nil, err
	}

	m, err := manifest.Load([]byte(r.Manifest), manifest.Environment(env))
	if err != nil {
		return nil, err
	}

	cs, err := p.CertificateList()
	if err != nil {
		return nil, err
	}

	ss := structs.Services{}

	for _, ms := range m.Services {
		cert := a.Outputs[fmt.Sprintf("Service%sCertificate", upperName(ms.Name))]
		cid := ""

		for _, c := range cs {
			if c.Arn == cert {
				cid = c.Id
			}
		}

		s := structs.Service{
			Name:   ms.Name,
			Domain: a.Outputs[fmt.Sprintf("Service%sEndpoint", upperName(ms.Name))],
			Ports: []structs.ServicePort{
				{Balancer: 80, Container: ms.Port.Port},
				{Balancer: 443, Container: ms.Port.Port, Certificate: cid},
			},
		}

		ss = append(ss, s)
	}

	return ss, nil
}

func (p *AWSProvider) serviceListGeneration1(a *structs.App) (structs.Services, error) {
	if a.Release == "" {
		return nil, fmt.Errorf("no release for app: %s", a.Name)
	}

	r, err := p.ReleaseGet(a.Name, a.Release)
	if err != nil {
		return nil, err
	}

	m, err := manifest1.Load([]byte(r.Manifest))
	if err != nil {
		return nil, err
	}

	ss := structs.Services{}

	for _, ms := range m.Services {
		s := structs.Service{
			Name:   ms.Name,
			Domain: a.Outputs[fmt.Sprintf("Balancer%sHost", upperName(ms.Name))],
			Ports:  []structs.ServicePort{},
		}

		for _, msp := range ms.Ports {
			p := structs.ServicePort{
				Balancer:  msp.Balancer,
				Container: msp.Container,
			}

			if lp := strings.Split(a.Parameters[fmt.Sprintf("%sPort%dListener", upperName(ms.Name), msp.Balancer)], ","); len(lp) > 1 {
				p.Certificate = certificateFriendlyId(lp[1])
			}

			s.Ports = append(s.Ports, p)
		}

		ss = append(ss, s)
	}

	return ss, nil
}

func (p *AWSProvider) ServiceUpdate(app, name string, port int, opts structs.ServiceUpdateOptions) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	switch a.Tags["Generation"] {
	case "", "1":
		return p.serviceUpdateGeneration1(a, name, port, opts)
	case "2":
	default:
		return fmt.Errorf("unknown generation for app: %s", app)
	}

	return fmt.Errorf("not yet supported for generation 2")

	return nil
}

func (p AWSProvider) serviceUpdateGeneration1(a *structs.App, name string, port int, opts structs.ServiceUpdateOptions) error {
	params := map[string]string{}

	if opts.Certificate != "" {
		cs, err := p.CertificateList()
		if err != nil {
			return err
		}

		for _, c := range cs {
			if c.Id == opts.Certificate {
				param := fmt.Sprintf("%sPort%dListener", upperName(name), port)
				fp := strings.Split(a.Parameters[param], ",")
				params[param] = fmt.Sprintf("%s,%s", fp[0], c.Arn)
			}
		}
	}

	return p.updateStack(p.rackStack(a.Name), "", params)
}
