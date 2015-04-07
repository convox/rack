package models

import "github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/ec2"

type AvailabilityZone struct {
	Name string
}

type AvailabilityZones []AvailabilityZone

func ListAvailabilityZones() (AvailabilityZones, error) {
	req := &ec2.DescribeAvailabilityZonesRequest{}

	res, err := EC2.DescribeAvailabilityZones(req)

	if err != nil {
		return nil, err
	}

	azs := make(AvailabilityZones, len(res.AvailabilityZones))

	for i, az := range res.AvailabilityZones {
		if i >= 3 {
			break
		}

		azs[i] = AvailabilityZone{
			Name: *az.ZoneName,
		}
	}

	return azs, nil
}

func (aa AvailabilityZones) Names() []string {
	azs := make([]string, len(aa))

	for i, a := range aa {
		azs[i] = a.Name
	}

	return azs
}
