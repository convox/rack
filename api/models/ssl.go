package models

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
)

type SSL struct {
	Id   string `json:"id"`
	Port int64  `json:"port"`
}

type SSLs []SSL

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
				Port: *listener.LoadBalancerPort,
			}
			ssls = append(ssls, ssl)
		}
	}

	return ssls, nil
}
