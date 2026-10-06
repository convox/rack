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

var classicBalancerACMKeyAlgorithms = map[string]bool{
	acm.KeyAlgorithmRsa1024: true,
	acm.KeyAlgorithmRsa2048: true,
}

func (p *Provider) certificateApplyGeneration1(a *structs.App, service string, port int, id string) error {
	cs, err := p.CertificateList()
	if err != nil {
		return err
	}

	var cert *structs.Certificate

	for i := range cs {
		if cs[i].Id == id {
			cert = &cs[i]
		}
	}

	if cert == nil {
		return fmt.Errorf("certificate not found")
	}

	if a.Parameters[fmt.Sprintf("Balancer%sType", upperName(service))] != "ALB" && strings.Contains(cert.Arn, ":acm:") {
		alg, err := p.certificateKeyAlgorithm(cert.Arn)
		if err != nil {
			return err
		}

		if alg != "" && !classicBalancerACMKeyAlgorithms[alg] {
			return fmt.Errorf("certificate %s is %s, which a Classic Load Balancer does not accept from ACM", id, alg)
		}
	}

	param := fmt.Sprintf("%sPort%dListener", upperName(service), port)
	fp := strings.Split(a.Parameters[param], ",")

	return p.updateStack(p.rackStack(a.Name), nil, map[string]string{param: fmt.Sprintf("%s,%s", fp[0], cert.Arn)}, map[string]string{}, "")
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

	p.waitForCertificateListed(*res.CertificateArn)

	return c, nil
}

func (p *Provider) CertificateDelete(id string) error {
	if strings.HasPrefix(id, "acm-") {
		id = strings.Split(id, "-")[1]

		certs, err := p.certificateListACM(acm.KeyAlgorithm_Values())
		if err != nil {
			return err
		}

		for _, c := range certs {
			if strings.HasSuffix(*c.CertificateArn, "-"+id) {
				_, err = p.acm().DeleteCertificate(&acm.DeleteCertificateInput{
					CertificateArn: c.CertificateArn,
				})
				if awsError(err) == "ResourceNotFoundException" {
					return fmt.Errorf("certificate not found")
				}
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
		DomainName:       aws.String(domains[0]),
		IdempotencyToken: aws.String(randomString(32)),
	}

	if len(alts) > 0 {
		req.SubjectAlternativeNames = alts
	}

	res, err := p.acm().RequestCertificate(req)

	if err != nil {
		return nil, err
	}

	p.waitForCertificateListed(*res.CertificateArn)

	parts := strings.Split(*res.CertificateArn, "-")
	id := fmt.Sprintf("acm-%s", parts[len(parts)-1])

	cert := structs.Certificate{
		Id:     id,
		Domain: domains[0],
	}

	return &cert, nil
}

func (p *Provider) CertificateList() (structs.Certificates, error) {
	return p.certificateList(acm.KeyAlgorithm_Values())
}

func (p *Provider) certificateList(keyTypes []string) (structs.Certificates, error) {
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

	ares, err := p.certificateListACM(keyTypes)
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
			tags[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
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

func (p *Provider) releaseCertificates() (structs.Certificates, error) {
	cs, err := p.certificateList(nil)
	if err != nil {
		return nil, err
	}

	ccs := structs.Certificates{}

	for _, c := range cs {
		if c.Expiration.After(time.Now()) {
			ccs = append(ccs, c)
		}
	}

	return ccs, nil
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

	c.Domain = aws.StringValue(res.Certificate.DomainName)
	c.Domains = make([]string, len(res.Certificate.SubjectAlternativeNames))

	for i, san := range res.Certificate.SubjectAlternativeNames {
		c.Domains[i] = aws.StringValue(san)
	}

	return c, nil
}

func (p *Provider) certificateKeyAlgorithm(arn string) (string, error) {
	res, err := p.acm().DescribeCertificate(&acm.DescribeCertificateInput{
		CertificateArn: aws.String(arn),
	})
	if err != nil {
		return "", err
	}

	if res.Certificate == nil {
		return "", nil
	}

	return strings.ReplaceAll(aws.StringValue(res.Certificate.KeyAlgorithm), "-", "_"), nil
}

func (p *Provider) certificateListACM(keyTypes []string) ([]*acm.CertificateSummary, error) {
	certs := []*acm.CertificateSummary{}

	req := &acm.ListCertificatesInput{}

	if len(keyTypes) > 0 {
		req.Includes = &acm.Filters{KeyTypes: aws.StringSlice(keyTypes)}
	}

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

var (
	certificateListWaitConfirmations = 2
	certificateListWaitTick          = 2 * time.Second
	certificateListWaitTimeout       = 10 * time.Second
)

func (p *Provider) waitForCertificateListed(arn string) {
	seen := 0
	done := time.Now().Add(certificateListWaitTimeout)

	for {
		cs, err := p.certificateListACM(acm.KeyAlgorithm_Values())
		if err == nil && certificateListed(cs, arn) {
			seen++
		} else {
			seen = 0
		}

		if seen >= certificateListWaitConfirmations || time.Now().After(done) {
			return
		}

		time.Sleep(certificateListWaitTick)
	}
}

func certificateListed(cs []*acm.CertificateSummary, arn string) bool {
	for _, c := range cs {
		if aws.StringValue(c.CertificateArn) == arn {
			return true
		}
	}

	return false
}
