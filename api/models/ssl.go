package models

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
)

type SSL struct {
	Id   string `json:"id"`
	Port string `json:"port"`
	Arn  string `json:"arn"`
}

type SSLs []SSL

func CreateSSL(a, balancerPort, body, key string) (*SSL, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

	// validate app has hostPort
	release, err := app.LatestRelease()

	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(release.Manifest)

	if err != nil {
		return nil, err
	}

	// TODO: validate based on manifest defining EXTERNAL port
	me := manifest.EntryByBalancerPort(balancerPort)

	if me == nil {
		return nil, fmt.Errorf("Manifest does not specify balancer port %s", balancerPort)
	}

	// upload certificate
	resp, err := IAM().UploadServerCertificate(&iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(body),
		PrivateKey:            aws.String(key),
		ServerCertificateName: aws.String(fmt.Sprintf("%s-%s", a, balancerPort)),
	})

	if err != nil {
		return nil, err
	}

	arn := resp.ServerCertificateMetadata.Arn
	name := resp.ServerCertificateMetadata.ServerCertificateName

	tmpl, err := release.Formation()

	if err != nil {
		return nil, err
	}

	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(app.Name),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		TemplateBody: aws.String(tmpl),
	}

	params := app.Parameters

	// TODO: make name (e.g.) WebPort3001Balancer based on EXTERNAL PORT and Manifest Process Name
	// WebPort3001Balancer, WebPort3001Certificate
	params[fmt.Sprintf("%sPort%sBalancer", UpperName(me.Name), balancerPort)] = balancerPort // e.g.WebPort3000Certificate
	params[fmt.Sprintf("%sPort%sCertificate", UpperName(me.Name), balancerPort)] = *arn      // e.g.WebPort3000Certificate

	for key, val := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(val),
		})
	}

	fmt.Printf("%+v\n", req.Parameters)
	fmt.Printf("%s\n", *req.TemplateBody)

	_, err = CloudFormation().UpdateStack(req)

	if err != nil {
		return nil, err
	}

	ssl := SSL{
		Id:   *name,
		Port: balancerPort,
		Arn:  *arn,
	}

	return &ssl, nil
}

func DeleteSSL(a, balancerPort string) (*SSL, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

	// validate app stack has certificate

	param := ""

	for k, v := range app.Parameters {
		if strings.HasSuffix(k, fmt.Sprintf("%sCertificate", balancerPort)) {
			if v != "" {
				param = k
			}
		}
	}

	if param == "" {
		return nil, fmt.Errorf("Stack does not have a Certificate on Balancer port %s", balancerPort)
	}

	changes := map[string]string{}
	changes[param] = ""

	app.UpdateParams(changes)

	// TODO: wait for stack update to finish, so we can delete certificate

	params := &iam.DeleteServerCertificateInput{
		ServerCertificateName: aws.String(app.Name),
	}

	resp, err := IAM().DeleteServerCertificate(params)

	if err != nil {
		return nil, err
	}

	fmt.Printf("%+v\n", resp)
	return nil, nil
}

func ListSSLs(a string) (SSLs, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

	ssls := make(SSLs, 0)

	fmt.Printf("%+v\n", app.Parameters)

	for k, v := range app.Parameters {
		if strings.HasSuffix(k, fmt.Sprintf("Certificate")) {
			if v != "" {
				ssls = append(ssls, SSL{
					Arn:  v,
					Port: "5000",
				})
			}
		}
	}

	return ssls, nil
}
