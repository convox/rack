package models

import (
	"fmt"
	"strings"
	"time"

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

	// validate app is not currently updating
	if app.Status != "running" {
		return nil, fmt.Errorf("can not update app with status: %s", app.Status)
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

	params[fmt.Sprintf("%sPort%sBalancer", UpperName(me.Name), balancerPort)] = balancerPort // e.g.WebPort443Balancer = 443
	params[fmt.Sprintf("%sPort%sCertificate", UpperName(me.Name), balancerPort)] = *arn      // e.g.WebPort443Certificate = arn:...

	for key, val := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(val),
		})
	}

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

	// validate app is not currently updating
	if app.Status != "running" {
		return nil, fmt.Errorf("can not update app with status: %s", app.Status)
	}

	// validate app stack has certificate
	param := ""
	arn := ""

	for k, v := range app.Parameters {
		if strings.HasSuffix(k, fmt.Sprintf("%sCertificate", balancerPort)) {
			if v != "" {
				arn = v
				param = k
			}
		}
	}

	if param == "" {
		return nil, fmt.Errorf("app does not have a certificate on balancer port %s", balancerPort)
	}

	changes := map[string]string{}
	changes[param] = ""

	app.UpdateParams(changes)

	go func() {
		for {
			time.Sleep(2 * time.Second)

			a, err := GetApp(app.Name)
			fmt.Printf("%+v\n%+v\n", a, err)

			if err != nil {
				return
			}

			if a.Status == "running" {
				params := &iam.DeleteServerCertificateInput{
					ServerCertificateName: aws.String(fmt.Sprintf("%s-%s", app.Name, balancerPort)),
				}

				resp, err := IAM().DeleteServerCertificate(params)
				fmt.Printf("%+v\n%+v\n", resp, err)

				return
			}
		}
	}()

	ssl := SSL{
		Port: balancerPort,
		Arn:  arn,
	}

	return &ssl, nil
}

func ListSSLs(a string) (SSLs, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

	ssls := make(SSLs, 0)

	for k, v := range app.Parameters {
		if strings.HasSuffix(k, fmt.Sprintf("Certificate")) {
			if v != "" {
				ssls = append(ssls, SSL{
					Arn:  v,
					Port: k,
				})
			}
		}
	}

	return ssls, nil
}
