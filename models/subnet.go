package models

import "fmt"

type Subnet struct {
	AvailabilityZone string
	Cidr             string
}

type Subnets []Subnet

func ListSubnets() (Subnets, error) {
	azs, err := ListAvailabilityZones()

	if err != nil {
		return nil, err
	}

	subnets := make(Subnets, len(azs))

	for i, az := range azs {
		subnets[i] = Subnet{
			AvailabilityZone: az.Name,
			Cidr:             fmt.Sprintf("10.0.%d.0/24", i+1),
		}
	}

	return subnets, nil
}
