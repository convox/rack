package handler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
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

func HandleEC2NatGateway(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING NATGATEWAY")
		fmt.Printf("req %+v\n", req)
		return EC2NatGatewayCreate(req)
	case "Update":
		fmt.Println("UPDATING NATGATEWAY")
		fmt.Printf("req %+v\n", req)
		return EC2NatGatewayUpdate(req)
	case "Delete":
		fmt.Println("DELETING NATGATEWAY")
		fmt.Printf("req %+v\n", req)
		return EC2NatGatewayDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

var regexMatchAvailabilityZones = regexp.MustCompile(`following availability zones: ([^.]+)`)

func EC2AvailabilityZonesCreate(req Request) (string, map[string]string, error) {
	_, err := EC2(req).CreateSubnet(&ec2.CreateSubnetInput{
		AvailabilityZone: aws.String("garbage"),
		CidrBlock:        aws.String("10.200.0.0/16"),
		VpcId:            aws.String(req.ResourceProperties["Vpc"].(string)),
	})

	matches := regexMatchAvailabilityZones.FindStringSubmatch(err.Error())
	matches = strings.Split(strings.Replace(matches[1], " ", "", -1), ",")

	if len(matches) < 1 {
		return "", nil, fmt.Errorf("could not discover availability zones")
	}

	outputs := make(map[string]string)
	for i, az := range matches {
		outputs["AvailabilityZone"+strconv.Itoa(i)] = az
	}

	physical := strings.Join(matches, ",")
	return physical, outputs, nil
}

func EC2AvailabilityZonesUpdate(req Request) (string, map[string]string, error) {
	// nop
	return req.PhysicalResourceId, nil, nil
}

func EC2AvailabilityZonesDelete(req Request) (string, map[string]string, error) {
	// nop
	return req.PhysicalResourceId, nil, nil
}

func EC2NatGatewayCreate(req Request) (string, map[string]string, error) {
	res, err := EC2(req).CreateNatGateway(&ec2.CreateNatGatewayInput{
		AllocationId: aws.String(req.ResourceProperties["AllocationId"].(string)),
		SubnetId:     aws.String(req.ResourceProperties["SubnetId"].(string)),
	})

	if err != nil {
		return "invalid", nil, err
	}

	return *res.NatGateway.NatGatewayId, nil, nil
}

func EC2NatGatewayUpdate(req Request) (string, map[string]string, error) {
	return req.PhysicalResourceId, nil, fmt.Errorf("could not update")
}

func EC2NatGatewayDelete(req Request) (string, map[string]string, error) {
	_, err := EC2(req).DeleteNatGateway(&ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(req.PhysicalResourceId),
	})

	return req.PhysicalResourceId, nil, err
}
