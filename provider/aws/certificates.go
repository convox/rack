package aws

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/pkg/cache"
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
	cs, err := p.certificateList(acm.KeyAlgorithm_Values())
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

		certs, err := p.certificateListACM(context.Background(), acm.KeyAlgorithm_Values(), nil)
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

func (p *Provider) CertificateGenerate(domains []string, opts structs.CertificateGenerateOptions) (*structs.Certificate, error) {
	if len(domains) < 1 {
		return nil, fmt.Errorf("must specify at least one domain")
	}

	dns := false

	// Email unless the client asks for DNS: an older CLI cannot show the records a DNS certificate waits on.
	switch strings.ToLower(aws.StringValue(opts.Validation)) {
	case "", "email":
	case "dns":
		dns = true
	default:
		return nil, fmt.Errorf("validation must be dns or email")
	}

	alts := []*string{}

	for _, domain := range domains[1:] {
		alts = append(alts, aws.String(domain))
	}

	req := &acm.RequestCertificateInput{
		DomainName:       aws.String(domains[0]),
		IdempotencyToken: aws.String(randomString(32)),
	}

	if dns {
		req.Tags = []*acm.Tag{
			{Key: aws.String("System"), Value: aws.String("convox")},
			{Key: aws.String("Rack"), Value: aws.String(p.Rack)},
		}
		req.ValidationMethod = aws.String(acm.ValidationMethodDns)
	}

	if len(alts) > 0 {
		req.SubjectAlternativeNames = alts
	}

	res, err := p.acm().RequestCertificate(req)

	if err != nil {
		return nil, err
	}

	p.waitForCertificateListed(*res.CertificateArn)

	if dns {
		if _, err := p.certificateRecordsWait(p.Context(), *res.CertificateArn, certificateGenerateWait); err != nil {
			fmt.Printf("ns=certificates at=generate arn=%s error=%q\n", *res.CertificateArn, err)
		}
	}

	cert := structs.Certificate{
		Id:     acmCertificateId(*res.CertificateArn),
		Domain: domains[0],
		Owned:  dns,
		Status: statusPending,
	}

	return &cert, nil
}

var certificateGenerateWait = 20 * time.Second

func (p *Provider) CertificateList() (structs.Certificates, error) {
	certs, err := p.iamCertificates()
	if err != nil {
		return nil, err
	}

	for i := range certs {
		certs[i].Status = statusImported
	}

	own, err := p.certificateOwnership()
	if err != nil {
		return nil, err
	}

	acs, err := p.acmCertificates(p.Context(), acmReadOptions{})
	if err != nil {
		return nil, err
	}

	for _, c := range acs {
		if c.err != nil {
			fmt.Printf("ns=certificates at=list id=%s error=%q\n", c.id, c.err)
			continue
		}

		if c.lookup() {
			continue
		}

		certs = append(certs, structs.Certificate{
			Arn:        c.arn,
			Id:         c.id,
			Domain:     c.domain,
			Domains:    c.sans,
			Expiration: c.expiration,
			Owned:      own.owns(c, p.Rack),
			Status:     c.statusValue(),
		})
	}

	return certs, nil
}

func (p *Provider) iamCertificates() (structs.Certificates, error) {
	certs := structs.Certificates{}

	req := &iam.ListServerCertificatesInput{}

	for {
		res, err := p.iam().ListServerCertificates(req)
		if err != nil {
			return nil, err
		}

		for _, cert := range res.ServerCertificateMetadataList {
			v, err := cache.Fetch("iamServerCertificate", aws.StringValue(cert.ServerCertificateId), 24*time.Hour, func() (interface{}, bool, error) {
				var res *iam.GetServerCertificateOutput

				err := retry(5, 2*time.Second, func() error {
					var err error

					res, err = p.iam().GetServerCertificate(&iam.GetServerCertificateInput{
						ServerCertificateName: cert.ServerCertificateName,
					})

					return err
				})

				return res, !p.SkipCache, err
			})
			if err != nil {
				return nil, err
			}

			sc, _ := v.(*iam.GetServerCertificateOutput)

			pem, _ := pem.Decode([]byte(*sc.ServerCertificate.CertificateBody))
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

	return certs, nil
}

// certificateList is the list templates and listeners may use: issued, not created by an app
// stack, and never a lookup.
func (p *Provider) certificateList(keyTypes []string) (structs.Certificates, error) {
	certs, err := p.iamCertificates()
	if err != nil {
		return nil, err
	}

	ares, err := p.certificateListACM(context.Background(), keyTypes, []string{acm.CertificateStatusIssued})
	if err != nil {
		return nil, err
	}

	for _, cert := range ares {
		usable, err := p.certificateUsable(*cert.CertificateArn)
		if err != nil {
			return nil, err
		}
		if !usable {
			continue
		}

		c := certificateFromSummary(cert)

		if c == nil {
			c, err = p.certificateGetACM(*cert.CertificateArn)
			if err != nil {
				return nil, err
			}
		}

		if c != nil {
			certs = append(certs, *c)
		}
	}

	return certs, nil
}

// The tags are set when the certificate is created and never change, so the verdict is cached.
func (p *Provider) certificateUsable(arn string) (bool, error) {
	v, err := cache.Fetch("certificateUsable", arn, 24*time.Hour, func() (interface{}, bool, error) {
		res, err := p.acm().ListTagsForCertificate(&acm.ListTagsForCertificateInput{
			CertificateArn: aws.String(arn),
		})
		if awsError(err) == "ResourceNotFoundException" {
			return false, false, nil
		}
		if err != nil {
			return nil, false, err
		}

		tags := map[string]string{}

		for _, t := range res.Tags {
			tags[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}

		excluded := (tags["System"] == "convox" && tags["Type"] == "app") || tags["Purpose"] == lookupPurpose

		return !excluded, !p.SkipCache, nil
	})
	if err != nil {
		return false, err
	}

	usable, _ := v.(bool)

	return usable, nil
}

func certificateFromSummary(s *acm.CertificateSummary) *structs.Certificate {
	domain := aws.StringValue(s.DomainName)
	sans := aws.StringValueSlice(s.SubjectAlternativeNameSummaries)

	if aws.BoolValue(s.HasAdditionalSubjectAlternativeNames) || s.NotAfter == nil || !slices.Contains(sans, domain) {
		return nil
	}

	return &structs.Certificate{
		Arn:        *s.CertificateArn,
		Id:         acmCertificateId(*s.CertificateArn),
		Domain:     domain,
		Domains:    sans,
		Expiration: *s.NotAfter,
	}
}

func (p *Provider) certificateID(arn string) (string, error) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return "", nil
	}

	switch parts[2] {
	case "acm":
		usable, err := p.certificateUsable(arn)
		if err != nil || !usable {
			return "", err
		}

		res, err := p.acm().DescribeCertificate(&acm.DescribeCertificateInput{
			CertificateArn: aws.String(arn),
		})
		if awsError(err) == "ResourceNotFoundException" {
			return "", nil
		}
		if err != nil {
			return "", err
		}

		if res.Certificate == nil || aws.StringValue(res.Certificate.Status) != acm.CertificateStatusIssued {
			return "", nil
		}

		return acmCertificateId(arn), nil
	case "iam":
		name := arn[strings.LastIndex(arn, "/")+1:]

		v, err := cache.Fetch("iamServerCertificateArn", name, 24*time.Hour, func() (interface{}, bool, error) {
			res, err := p.iam().GetServerCertificate(&iam.GetServerCertificateInput{
				ServerCertificateName: aws.String(name),
			})
			if awsError(err) == "NoSuchEntity" {
				return "", false, nil
			}
			if err != nil {
				return nil, false, err
			}

			return aws.StringValue(res.ServerCertificate.ServerCertificateMetadata.Arn), !p.SkipCache, nil
		})
		if err != nil {
			return "", err
		}

		if found, _ := v.(string); found == arn {
			return name, nil
		}
	}

	return "", nil
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

func (p *Provider) certificateListACM(ctx context.Context, keyTypes, statuses []string) ([]*acm.CertificateSummary, error) {
	certs := []*acm.CertificateSummary{}

	req := &acm.ListCertificatesInput{}

	if len(keyTypes) > 0 {
		req.Includes = &acm.Filters{KeyTypes: aws.StringSlice(keyTypes)}
	}

	if len(statuses) > 0 {
		req.CertificateStatuses = aws.StringSlice(statuses)
	}

	for {
		res, err := p.acm().ListCertificatesWithContext(ctx, req)
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
	ctx, cancel := context.WithTimeout(p.Context(), certificateListWaitTimeout)
	defer cancel()

	seen := 0

	for {
		cs, err := p.certificateListACM(ctx, acm.KeyAlgorithm_Values(), nil)
		if err == nil && certificateListed(cs, arn) {
			seen++
		} else {
			seen = 0
		}

		if seen >= certificateListWaitConfirmations {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(certificateListWaitTick):
		}
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
