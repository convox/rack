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

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/iam"
)

type SSL struct {
	Expiration time.Time `json:"expiration"`
	Domain     string    `json:"domain"`
	Process    string    `json:"process"`
	Port       int       `json:"port"`
	Secure     bool      `json:"secure"`
}

type SSLs []SSL

func CreateSSL(app, process string, port int, arn string, body, key string, chain string, secure bool) (*SSL, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	// validate app is not currently updating
	if a.Status != "running" {
		return nil, fmt.Errorf("can not update app with status: %s", a.Status)
	}

	// validate app has hostPort
	release, err := a.LatestRelease()

	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(release.Manifest, a)

	if err != nil {
		return nil, err
	}

	me := manifest.Entry(process)

	if me == nil {
		return nil, fmt.Errorf("no such process: %s", process)
	}

	found := false

	for _, p := range me.ExternalPorts() {
		if strings.HasPrefix(p, fmt.Sprintf("%d:", port)) {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("process does not expose port: %d", port)
	}

	if arn == "" {
		arn, err = uploadCert(a, process, port, body, key, chain)

		if err != nil {
			return nil, err
		}
	}
	// Check that the arn is valid?

	tmpl, err := release.Formation()

	if err != nil {
		return nil, err
	}

	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(a.StackName()),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		TemplateBody: aws.String(tmpl),
	}

	params := a.Parameters

	params[fmt.Sprintf("%sPort%dCertificate", UpperName(me.Name), port)] = arn // e.g.WebPort443Certificate = arn:...

	if secure {
		params[fmt.Sprintf("%sPort%dSecure", UpperName(me.Name), port)] = "Yes"
	}

	for key, val := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(val),
		})
	}

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

func DeleteSSL(app, process string, port int) (*SSL, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	// validate app is not currently updating
	if a.Status != "running" {
		return nil, fmt.Errorf("can not update app with status: %s", a.Status)
	}

	param := fmt.Sprintf("%sPort%dCertificate", UpperName(process), port)

	arn := a.Parameters[param]

	if arn == "" {
		return nil, fmt.Errorf("could not find target")
	}

	changes := map[string]string{}
	changes[param] = ""

	secureParam := fmt.Sprintf("%sPort%dSecure", UpperName(process), port)

	if a.Parameters[secureParam] == "Yes" {
		a.Parameters[secureParam] = "No"
	}

	a.UpdateParams(changes)

	go func() {
		for {
			time.Sleep(5 * time.Second)

			a, err := GetApp(a.Name)

			if err != nil {
				return
			}

			if a.Status == "running" {
				params := &iam.DeleteServerCertificateInput{
					ServerCertificateName: aws.String(certName(a.StackName(), process, port)),
				}

				resp, err := IAM().DeleteServerCertificate(params)
				fmt.Printf("%+v\n%+v\n", resp, err)

				return
			}
		}
	}()

	ssl := SSL{
		Port:    port,
		Process: process,
	}

	return &ssl, nil
}

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

			resp, err := IAM().GetServerCertificate(&iam.GetServerCertificateInput{
				ServerCertificateName: aws.String(certName(app.StackName(), matches[1], port)),
			})

			if err != nil {
				return nil, err
			}

			pemBlock, _ := pem.Decode([]byte(*resp.ServerCertificate.CertificateBody))
			c, err := x509.ParseCertificate(pemBlock.Bytes)

			secure := app.Parameters[fmt.Sprintf("%sPort%sSecure", matches[1], matches[2])] == "Yes"

			ssls = append(ssls, SSL{
				Domain:     c.Subject.CommonName,
				Expiration: *resp.ServerCertificate.ServerCertificateMetadata.Expiration,
				Port:       port,
				Process:    DashName(matches[1]),
				Secure:     secure,
			})
		}
	}

	return ssls, nil
}

func UpdateSSL(app, process string, port int, body, key string, chain string) (*SSL, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	// validate app is not currently updating
	if a.Status != "running" {
		return nil, fmt.Errorf("can not update app with status: %s", a.Status)
	}

	// store old cert name
	oldCertName := certName(app, process, port)

	// validate process exists
	if oldCertName == "" {
		return nil, fmt.Errorf("no certificate configured for %s port %d", process, port)
	}

	outputs := a.Outputs

	balancer := outputs[fmt.Sprintf("%sPort%dBalancerName", UpperName(process), port)]

	if balancer == "" {
		return nil, fmt.Errorf("Balancer ouptut not found. Please redeploy your app and try again.")
	}

	// upload new cert
	arn, err := uploadCert(a, process, port, body, key, chain)

	if err != nil {
		return nil, err
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
