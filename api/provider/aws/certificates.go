package aws

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) CertificateCreate(pub, key, chain string) (*structs.Certificate, error) {
	end, _ := pem.Decode([]byte(pub))
	pub = string(pem.EncodeToMemory(end))

	if chain == "" {
		ch, err := resolveCertificateChain(pub)
		if err != nil {
			return nil, err
		}

		chain = ch
	}

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
			Id:         *cert.ServerCertificateName,
			Domain:     c.Subject.CommonName,
			Expiration: *cert.Expiration,
		})
	}

	// only fetch ACM certificates in regions that support it
	switch os.Getenv("AWS_REGION") {
	case "us-east-1":
		c, err := p.certificateListACM()

		if err != nil {
			return nil, err
		}

		certs = append(certs, c...)
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

func (p *AWSProvider) certificateListACM() (structs.Certificates, error) {
	certs := structs.Certificates{}

	ares, err := p.acm().ListCertificates(nil)

	if err != nil {
		return nil, err
	}

	for _, cert := range ares.CertificateSummaryList {
		parts := strings.Split(*cert.CertificateArn, "-")
		id := fmt.Sprintf("acm-%s", parts[len(parts)-1])

		c := structs.Certificate{
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

// use cfssl bundle to generate the certificate chain
// return the whole list minus the first one
func resolveCertificateChain(body string) (string, error) {
	bl, _ := pem.Decode([]byte(body))
	crt, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		return "", err
	}

	// return if this is a self-signed cert
	// a cert is self-signed if the issuer and subject are the same
	if string(crt.RawIssuer) == string(crt.RawSubject) {
		return "", nil
	}

	// return if this is a cloudflare origin cert
	ou := crt.Issuer.OrganizationalUnit
	if len(ou) == 1 && ou[0] == "CloudFlare Origin SSL Certificate Authority" {
		return "", nil
	}

	cmd := exec.Command("cfssl", "bundle", "-cert", "-")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	stdin.Write([]byte(body))
	stdin.Close()

	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	edata, err := ioutil.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	fmt.Printf("cfssl stderr=%q\n", edata)

	// try to coerce last line of stderr into a friendly error message
	if len(data) == 0 && len(edata) > 0 {
		lines := strings.Split(strings.TrimSpace(string(edata)), "\n")
		l := lines[len(lines)-1]

		var e CfsslError

		err = json.Unmarshal([]byte(l), &e)
		if err != nil {
			return "", err
		}

		return "", e
	}

	var bundle CfsslCertificateBundle

	err = json.Unmarshal(data, &bundle)
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	certs := []*x509.Certificate{}

	raw := []byte(bundle.Bundle)

	for {
		block, rest := pem.Decode(raw)

		if block == nil {
			break
		}

		raw = rest

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return "", nil
		}

		certs = append(certs, cert)
	}

	var buf bytes.Buffer

	for i := 1; i < len(certs); i++ {
		err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: certs[i].Raw})
		if err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}
