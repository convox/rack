package models

import (
	"fmt"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/ec2"
)

type Subnet struct {
	AvailabilityZone string
	Cidr             string
}

type Subnets []Subnet

func ListSubnets() (Subnets, error) {
	subnets := Subnets{}

	req := &ec2.DescribeAvailabilityZonesRequest{}

	res, err := EC2.DescribeAvailabilityZones(req)

	if err != nil {
		return nil, err
	}

	for i, az := range res.AvailabilityZones {
		subnets = append(subnets, Subnet{
			AvailabilityZone: *az.ZoneName,
			Cidr:             fmt.Sprintf("10.0.%d.0/24", i+1),
		})
	}

	return subnets, nil
}
