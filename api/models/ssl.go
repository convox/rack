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
	_, err := GetApp(a)

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

	return &ssl, nil
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
