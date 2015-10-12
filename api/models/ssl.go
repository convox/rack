package models

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
)

type SSL struct {
	Id   string `json:"id"`
	Port string `json:"port"`
	Arn  string `json:"arn"`
}

type SSLs []SSL

func CreateSSL(a, port, body, key string) (*SSL, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

	params := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(body),
		PrivateKey:            aws.String(key),
		ServerCertificateName: aws.String(fmt.Sprintf("%s", a)),
	}

	resp, err := IAM().UploadServerCertificate(params)

	if err != nil {
		return nil, err
	}

	arn := resp.ServerCertificateMetadata.Arn
	name := resp.ServerCertificateMetadata.ServerCertificateName

	ssl := SSL{
		Id:   *name,
		Port: port,
		Arn:  *arn,
	}

	fmt.Println("%+v\n", app)
	stack_params := map[string]string{}
	stack_params[fmt.Sprintf("SSL%sArn", port)] = ssl.Arn
	stack_params[fmt.Sprintf("SSL%sPort", port)] = ssl.Port

	err = app.UpdateParams(stack_params)

	if err != nil {
		return nil, err
	}

	return &ssl, nil
}

func DeleteSSL(a string) (*SSL, error) {
	app, err := GetApp(a)

	if err != nil {
		return nil, err
	}

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

	resources := app.Resources()

	id := resources["Balancer"].Id

	params := elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{
			aws.String(id),
		},
	}

	resp, err := ELB().DescribeLoadBalancers(&params)

	if err != nil {
		return nil, err
	}

	lds := resp.LoadBalancerDescriptions[0].ListenerDescriptions

	ssls := make(SSLs, 0)

	for _, ld := range lds {
		listener := ld.Listener
		if listener.SSLCertificateId != nil {
			ssl := SSL{
				Id:   *listener.SSLCertificateId,
				Port: strconv.FormatInt(*listener.LoadBalancerPort, 10),
			}
			ssls = append(ssls, ssl)
		}
	}

	return ssls, nil
}
