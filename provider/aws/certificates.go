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
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) CertificateApply(app, service string, port int, id string) error {
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

	return fmt.Errorf("not yet implemented on generation 2")
}

func (p *AWSProvider) certificateApplyGeneration1(a *structs.App, service string, port int, id string) error {
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

	return p.updateStack(p.rackStack(a.Name), "", params)
}

func (p *AWSProvider) CertificateCreate(pub, key, chain string) (*structs.Certificate, error) {
	end, _ := pem.Decode([]byte(pub))
	pub = string(pem.EncodeToMemory(end))

	c, err := x509.ParseCertificate(end.Bytes)
	if err != nil {
		return nil, err
	}

	req := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(pub),
		PrivateKey:            aws.String(key),
		ServerCertificateName: aws.String(fmt.Sprintf("cert-%d", time.Now().Unix())),
	}

	if chain != "" {
		req.CertificateChain = aws.String(chain)
	}

	res, err := p.iam().UploadServerCertificate(req)

	if err != nil {
		return nil, err
	}

	parts := strings.Split(*res.ServerCertificateMetadata.Arn, "/")
	id := parts[len(parts)-1]

	cert := structs.Certificate{
		Id:         id,
		Domain:     c.Subject.CommonName,
		Expiration: *res.ServerCertificateMetadata.Expiration,
	}

	return &cert, nil
}

func (p *AWSProvider) CertificateDelete(id string) error {
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

func (p *AWSProvider) CertificateGenerate(domains []string) (*structs.Certificate, error) {
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

func (p *AWSProvider) CertificateList() (structs.Certificates, error) {
	res, err := p.iam().ListServerCertificates(nil)

	if err != nil {
		return nil, err
	}

	certs := structs.Certificates{}

	for _, cert := range res.ServerCertificateMetadataList {
		res, err := p.iam().GetServerCertificate(&iam.GetServerCertificateInput{
			ServerCertificateName: cert.ServerCertificateName,
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

	ares, err := p.certificateListACM()
	if err != nil {
		return nil, err
	}

	for _, cert := range ares {
		parts := strings.Split(*cert.CertificateArn, "-")
		id := fmt.Sprintf("acm-%s", parts[len(parts)-1])

		c := structs.Certificate{
			Arn:    *cert.CertificateArn,
			Id:     id,
			Domain: *cert.DomainName,
		}

		res, err := p.acm().DescribeCertificate(&acm.DescribeCertificateInput{
			CertificateArn: cert.CertificateArn,
		})
		if err != nil {
			return nil, err
		}

		if res.Certificate.NotAfter != nil {
			c.Expiration = *res.Certificate.NotAfter
		}

		certs = append(certs, c)
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

func (p *AWSProvider) certificateListACM() ([]*acm.CertificateSummary, error) {
	certs := []*acm.CertificateSummary{}
	input := &acm.ListCertificatesInput{}

	for {
		ares, err := p.acm().ListCertificates(input)
		if err != nil {
			return nil, err
		}

		certs = append(certs, ares.CertificateSummaryList...)

		if ares.NextToken == nil {
			return certs, nil
		}

		input.NextToken = ares.NextToken
	}
}
