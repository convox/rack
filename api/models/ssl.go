package models

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
)

type SSL struct {
	Certificate string    `json:"certificate"`
	Expiration  time.Time `json:"expiration"`
	Domain      string    `json:"domain"`
	Process     string    `json:"process"`
	Port        int       `json:"port"`
	Secure      bool      `json:"secure"`
}

type SSLs []SSL

func ListSSLs(a string) (SSLs, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

	ssls := make(SSLs, 0)

	// Find stack Parameters like WebPort443Certificate with an ARN set for the value
	// Get and decode corresponding certificate info
	re := regexp.MustCompile(`(\w+)Port(\d+)Certificate`)

	for k, v := range app.Parameters {
		if v == "" {
			continue
		}

		if matches := re.FindStringSubmatch(k); len(matches) > 0 {
			port, err := strconv.Atoi(matches[2])

			if err != nil {
				return nil, err
			}

			secure := app.Parameters[fmt.Sprintf("%sPort%sSecure", matches[1], matches[2])] == "Yes"

			switch prefix := v[8:11]; prefix {
			case "acm":
				res, err := ACM().DescribeCertificate(&acm.DescribeCertificateInput{
					CertificateArn: aws.String(v),
				})

				if err != nil {
					return nil, err
				}

				parts := strings.Split(v, "-")
				id := fmt.Sprintf("acm-%s", parts[len(parts)-1])

				ssls = append(ssls, SSL{
					Certificate: id,
					Domain:      *res.Certificate.DomainName,
					Expiration:  *res.Certificate.NotAfter,
					Port:        port,
					Process:     DashName(matches[1]),
					Secure:      secure,
				})
			case "iam":
				res, err := IAM().GetServerCertificate(&iam.GetServerCertificateInput{
					ServerCertificateName: aws.String(certName(app.StackName(), matches[1], port)),
				})

				if err != nil {
					return nil, err
				}

				pemBlock, _ := pem.Decode([]byte(*res.ServerCertificate.CertificateBody))

				c, err := x509.ParseCertificate(pemBlock.Bytes)

				if err != nil {
					return nil, err
				}

				ssls = append(ssls, SSL{
					Certificate: *res.ServerCertificate.ServerCertificateMetadata.ServerCertificateName,
					Domain:      c.Subject.CommonName,
					Expiration:  *res.ServerCertificate.ServerCertificateMetadata.Expiration,
					Port:        port,
					Process:     DashName(matches[1]),
					Secure:      secure,
				})
			default:
				return nil, fmt.Errorf("unknown arn prefix: %s", prefix)
			}
		}
	}

	return ssls, nil
}

func UpdateSSL(app, process string, port int, id string) (*SSL, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	// validate app is not currently updating
	if a.Status != "running" {
		return nil, fmt.Errorf("can not update app with status: %s", a.Status)
	}

	outputs := a.Outputs
	balancer := outputs[fmt.Sprintf("%sPort%dBalancerName", UpperName(process), port)]

	if balancer == "" {
		return nil, fmt.Errorf("Process and port combination unknown")
	}

	arn := ""

	if strings.HasPrefix(id, "acm-") {
		uuid := id[4:]

		res, err := ACM().ListCertificates(nil)

		if err != nil {
			return nil, err
		}

		for _, cert := range res.CertificateSummaryList {
			parts := strings.Split(*cert.CertificateArn, "-")

			if parts[len(parts)-1] == uuid {
				arn = *cert.CertificateArn
				break
			}
		}
	} else {
		res, err := IAM().GetServerCertificate(&iam.GetServerCertificateInput{
			ServerCertificateName: aws.String(id),
		})

		if err != nil {
			return nil, err
		}

		arn = *res.ServerCertificate.ServerCertificateMetadata.Arn
	}

	// update cloudformation
	req := &cloudformation.UpdateStackInput{
		StackName:           aws.String(a.StackName()),
		Capabilities:        []*string{aws.String("CAPABILITY_IAM")},
		UsePreviousTemplate: aws.Bool(true),
	}

	params := a.Parameters
	params[fmt.Sprintf("%sPort%dCertificate", UpperName(process), port)] = arn

	for key, val := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(val),
		})
	}

	// TODO: The existing cert will be orphaned. Deleting it now could cause
	// CF problems if the stack tries to rollback and use the old cert.
	_, err = UpdateStack(req)

	if err != nil {
		return nil, err
	}

	ssl := SSL{
		Port:    port,
		Process: process,
	}

	return &ssl, nil
}

// fetch certificate from CF params and parse name from arn
func certName(app, process string, port int) string {
	key := fmt.Sprintf("%sPort%dCertificate", UpperName(process), port)

	a, err := GetApp(app)

	if err != nil {
		fmt.Printf(err.Error())
		return ""
	}

	arn := a.Parameters[key]

	slice := strings.Split(arn, "/")

	return slice[len(slice)-1]
}

type CfsslCertificateBundle struct {
	Bundle string `json:"bundle"`
}

func deleteCert(certName string) error {
	_, err := IAM().DeleteServerCertificate(&iam.DeleteServerCertificateInput{
		ServerCertificateName: aws.String(certName),
	})

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
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

	cmd := exec.Command("cfssl", "bundle", "-cert", "-")

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()

	if err != nil {
		return "", err
	}

	stdout, err := cmd.StdoutPipe()

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

func uploadCert(a *App, process string, port int, body, key string, chain string) (string, error) {
	// strip off any intermediate certs from the body
	endEntityCert, _ := pem.Decode([]byte(body))
	body = string(pem.EncodeToMemory(endEntityCert))

	if chain == "" {
		var err error
		chain, err = resolveCertificateChain(body)

		if err != nil {
			return "", fmt.Errorf("could not generate chain: %s", err)
		}
	}

	// generate certificate name
	currentTime := time.Now()

	timestamp := currentTime.Format("20060102150405")

	name := fmt.Sprintf("%s%s%d-%s", UpperName(a.StackName()), UpperName(process), port, timestamp)

	input := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(body),
		PrivateKey:            aws.String(key),
		ServerCertificateName: aws.String(name),
	}

	// Only include chain if it's not an empty string
	if chain != "" {
		input.CertificateChain = aws.String(chain)
	}

	// upload certificate
	resp, err := IAM().UploadServerCertificate(input)

	if err != nil {
		return "", err
	}

	arn := resp.ServerCertificateMetadata.Arn

	return *arn, err
}
