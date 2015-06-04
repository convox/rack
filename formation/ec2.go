package formation

import (
	"fmt"
	"regexp"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
)

func HandleEC2AvailabilityZones(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING AVAILABILITYZONES")
		fmt.Printf("req %+v\n", req)
		return EC2AvailabilityZonesCreate(req)
	case "Update":
		fmt.Println("UPDATING AVAILABILITYZONES")
		fmt.Printf("req %+v\n", req)
		return EC2AvailabilityZonesUpdate(req)
	case "Delete":
		fmt.Println("DELETING AVAILABILITYZONES")
		fmt.Printf("req %+v\n", req)
		return EC2AvailabilityZonesDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

var regexMatchAvailabilityZones = regexp.MustCompile(`following availability zones: ([^,.]+), ([^,.]+), ([^,.]+)`)

func EC2AvailabilityZonesCreate(req Request) (string, map[string]string, error) {
	_, err := EC2(req).CreateSubnet(&ec2.CreateSubnetInput{
		AvailabilityZone: aws.String("garbage"),
		CIDRBlock:        aws.String("10.200.0.0/16"),
		VPCID:            aws.String(req.ResourceProperties["Vpc"].(string)),
	})

	matches := regexMatchAvailabilityZones.FindStringSubmatch(err.Error())

	if len(matches) != 4 {
		return "", nil, fmt.Errorf("could not discover availability zones")
	}

	outputs := map[string]string{
		"AvailabilityZone0": matches[1],
		"AvailabilityZone1": matches[2],
		"AvailabilityZone2": matches[3],
	}

	physical := fmt.Sprintf("%s,%s,%s", matches[1], matches[2], matches[3])

	return physical, outputs, nil
}

func EC2AvailabilityZonesUpdate(req Request) (string, map[string]string, error) {
	// nop
	return "", nil, nil
}

func EC2AvailabilityZonesDelete(req Request) (string, map[string]string, error) {
	// nop
	return "", nil, nil
}
