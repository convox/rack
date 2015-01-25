package models

type Subnet struct {
	AvailabilityZone string
	Cidr             string
}

type Subnets []Subnet

func ListSubnets() Subnets {
	return Subnets{
		Subnet{AvailabilityZone: "us-east-1a", Cidr: "10.0.1.0/24"},
		Subnet{AvailabilityZone: "us-east-1c", Cidr: "10.0.2.0/24"},
		Subnet{AvailabilityZone: "us-east-1d", Cidr: "10.0.3.0/24"},
		Subnet{AvailabilityZone: "us-east-1e", Cidr: "10.0.4.0/24"},
	}
}
