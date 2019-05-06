package aws

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) CertificateApply(app, service string, port int, id string) error {
	fmt.Printf("app = %+v\n", app)
	fmt.Printf("service = %+v\n", service)
	fmt.Printf("port = %+v\n", port)
	fmt.Printf("id = %+v\n", id)

	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	switch a.Tags["Generation"] {
	case "", "1":
		return p.certificateApplyGeneration1(a, service, port, id)
	case "2":
	default:
		return fmt.Errorf("unknown generation for app: %s", app)
	}

	return fmt.Errorf("generation 2 apps use the domain: attribute on services in convox.yml")
}

func (p *Provider) certificateApplyGeneration1(a *structs.App, service string, port int, id string) error {
	params := map[string]string{}

	cs, err := p.CertificateList()
	if err != nil {
		return err
	}

	for _, c := range cs {
		if c.Id == id {
			param := fmt.Sprintf("%sPort%dListener", upperName(service), port)
			fp := strings.Split(a.Parameters[param], ",")
			params[param] = fmt.Sprintf("%s,%s", fp[0], c.Arn)
		}
	}

	return p.updateStack(p.rackStack(a.Name), nil, params, map[string]string{}, "")
}

func (p *Provider) CertificateCreate(pub, key string, opts structs.CertificateCreateOptions) (*structs.Certificate, error) {
	req := &acm.ImportCertificateInput{
		Certificate: []byte(pub),
		PrivateKey:  []byte(key),
	}

	if opts.Chain != nil {
		req.CertificateChain = []byte(*opts.Chain)
	}

	res, err := p.acm().ImportCertificate(req)
	if err != nil {
		return nil, err
	}

	c, err := p.certificateGetACM(*res.CertificateArn)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *Provider) CertificateDelete(id string) error {
	if strings.HasPrefix(id, "acm") {
		ss := strings.Split(id, "-")
		if len(ss) < 2 {
			return fmt.Errorf("invalid certificate id")
		}
		id = ss[1]

		certs, err := p.certificateListACM()
		if err != nil {
			return err
		}

		for _, c := range certs {
			if strings.HasSuffix(*c.CertificateArn, id) {
				_, err = p.acm().DeleteCertificate(&acm.DeleteCertificateInput{
					CertificateArn: c.CertificateArn,
				})
				return err
			}
		}

		return fmt.Errorf("certificate not found")
	}

	_, err := p.iam().DeleteServerCertificate(&iam.DeleteServerCertificateInput{
		ServerCertificateName: aws.String(id),
	})

	return err
}

func (p *Provider) CertificateGenerate(domains []string) (*structs.Certificate, error) {
	if len(domains) < 1 {
		return nil, fmt.Errorf("must specify at least one domain")
	}

	alts := []*string{}

	for _, domain := range domains[1:] {
		alts = append(alts, aws.String(domain))
	}

	req := &acm.RequestCertificateInput{
		DomainName: aws.String(domains[0]),
	}

	if len(alts) > 0 {
		req.SubjectAlternativeNames = alts
	}

	res, err := p.acm().RequestCertificate(req)

	if err != nil {
		return nil, err
	}

	parts := strings.Split(*res.CertificateArn, "-")
	id := fmt.Sprintf("acm-%s", parts[len(parts)-1])

	cert := structs.Certificate{
		Id:     id,
		Domain: domains[0],
	}

	return &cert, nil
}

func (p *Provider) CertificateList() (structs.Certificates, error) {
	certs := structs.Certificates{}

	req := &iam.ListServerCertificatesInput{}

	for {
		res, err := p.iam().ListServerCertificates(req)
		if err != nil {
			return nil, err
		}

		for _, cert := range res.ServerCertificateMetadataList {
			var res *iam.GetServerCertificateOutput

			err = retry(5, 2*time.Second, func() error {
				res, err = p.iam().GetServerCertificate(&iam.GetServerCertificateInput{
					ServerCertificateName: cert.ServerCertificateName,
				})
				return err
			})
			if err != nil {
				return nil, err
			}

			pem, _ := pem.Decode([]byte(*res.ServerCertificate.CertificateBody))
			if err != nil {
				return nil, err
			}

			c, err := x509.ParseCertificate(pem.Bytes)
			if err != nil {
				return nil, err
			}

			certs = append(certs, structs.Certificate{
				Arn:        *cert.Arn,
				Id:         *cert.ServerCertificateName,
				Domain:     c.Subject.CommonName,
				Expiration: *cert.Expiration,
			})
		}

		if res.Marker == nil {
			break
		}

		req.Marker = res.Marker
	}

	ares, err := p.certificateListACM()
	if err != nil {
		return nil, err
	}

	for _, cert := range ares {
		tags := map[string]string{}

		tres, err := p.acm().ListTagsForCertificate(&acm.ListTagsForCertificateInput{
			CertificateArn: cert.CertificateArn,
		})
		if awsError(err) == "ResourceNotFoundException" {
			continue
		}
		if err != nil {
			return nil, err
		}

		for _, t := range tres.Tags {
			tags[*t.Key] = *t.Value
		}

		if tags["System"] == "convox" && tags["Type"] == "app" {
			continue
		}

		c, err := p.certificateGetACM(*cert.CertificateArn)
		if err != nil {
			return nil, err
		}

		if c != nil {
			certs = append(certs, *c)
		}
	}

	return certs, nil
}

type CfsslCertificateBundle struct {
	Bundle string `json:"bundle"`
}

type CfsslError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e CfsslError) Error() string {
	return e.Message
}

func (p *Provider) certificateGetACM(arn string) (*structs.Certificate, error) {
	parts := strings.Split(arn, "-")
	id := fmt.Sprintf("acm-%s", parts[len(parts)-1])

	c := &structs.Certificate{
		Arn: arn,
		Id:  id,
	}

	var res *acm.DescribeCertificateOutput
	var err error

	err = retry(5, 2*time.Second, func() error {
		res, err = p.acm().DescribeCertificate(&acm.DescribeCertificateInput{
			CertificateArn: aws.String(arn),
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	if *res.Certificate.Status != "ISSUED" {
		return nil, nil
	}

	if res.Certificate.NotAfter != nil {
		c.Expiration = *res.Certificate.NotAfter
	}

	c.Domain = *res.Certificate.DomainName
	c.Domains = make([]string, len(res.Certificate.SubjectAlternativeNames))

	for i, san := range res.Certificate.SubjectAlternativeNames {
		c.Domains[i] = *san
	}

	return c, nil
}

func (p *Provider) certificateListACM() ([]*acm.CertificateSummary, error) {
	certs := []*acm.CertificateSummary{}

	req := &acm.ListCertificatesInput{}

	for {
		res, err := p.acm().ListCertificates(req)
		if err != nil {
			return nil, err
		}

		certs = append(certs, res.CertificateSummaryList...)

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	return certs, nil
}
