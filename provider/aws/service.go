package aws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/convox/rack/helpers"
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

	env, err := helpers.AppEnvironment(p, app)
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

		parts := strings.SplitN(a.Parameters[fmt.Sprintf("%sFormation", upperName(ms.Name))], ",", 3)

		if len(parts) != 3 {
			return nil, fmt.Errorf("could not read formation for service: %s", ms.Name)
		}

		s.Count, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}

		s.Cpu, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}

		s.Memory, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
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

		parts := strings.SplitN(a.Parameters[fmt.Sprintf("%sFormation", upperName(ms.Name))], ",", 3)

		if len(parts) != 3 {
			return nil, fmt.Errorf("could not read formation for service: %s", ms.Name)
		}

		s.Count, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}

		s.Cpu, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}

		s.Memory, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
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

func (p *AWSProvider) ServiceUpdate(app, name string, opts structs.ServiceUpdateOptions) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	param := fmt.Sprintf("%sFormation", upperName(name))

	parts := strings.SplitN(a.Parameters[param], ",", 3)

	if len(parts) != 3 {
		return fmt.Errorf("could not read formation for service: %s", name)
	}

	if opts.Count != nil {
		parts[0] = strconv.Itoa(*opts.Count)
	}

	if opts.Cpu != nil {
		parts[1] = strconv.Itoa(*opts.Cpu)
	}

	if opts.Memory != nil {
		parts[2] = strconv.Itoa(*opts.Memory)
	}

	if err := p.updateStack(p.rackStack(a.Name), "", map[string]string{param: strings.Join(parts, ",")}); err != nil {
		return err
	}

	return nil
}
