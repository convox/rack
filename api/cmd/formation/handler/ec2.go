package handler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
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

func HandleEC2Route(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING ROUTE")
		fmt.Printf("req %+v\n", req)
		return EC2RouteCreate(req)
	case "Update":
		fmt.Println("UPDATING ROUTE")
		fmt.Printf("req %+v\n", req)
		return EC2RouteUpdate(req)
	case "Delete":
		fmt.Println("DELETING ROUTE")
		fmt.Printf("req %+v\n", req)
		return EC2RouteDelete(req)
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
	azs := strings.Split(req.PhysicalResourceId, ",")

	outputs := make(map[string]string)
	for i, az := range azs {
		outputs["AvailabilityZone"+strconv.Itoa(i)] = az
	}

	// nop
	return req.PhysicalResourceId, outputs, nil
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

	// block for 2 minutes until it's deleted
	// Fixes subsequent CF error on deleting Elastic IP:
	//   API: ec2:disassociateAddress You do not have permission to access the specified resource.
	for i := 0; i < 12; i++ {
		resp, derr := EC2(req).DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{aws.String(req.PhysicalResourceId)},
		})

		if derr != nil {
			fmt.Printf("EC2NatGatewayDelete error: %s\n", derr)

			// if nat gateway not found, break
			if ae, ok := derr.(awserr.Error); ok {
				if ae.Code() == "InvalidParameterException" {
					break
				}
			}
		}

		// if NAT gateway is deleted, break
		if len(resp.NatGateways) == 1 {
			n := resp.NatGateways[0]

			if *n.State == "deleted" {
				break
			}
		}

		// sleep and retry
		time.Sleep(10 * time.Second)
	}

	// return original DeleteNatGateway success / failure
	return req.PhysicalResourceId, nil, err
}

func EC2RouteCreate(req Request) (string, map[string]string, error) {
	destinationCidrBlock := req.ResourceProperties["DestinationCidrBlock"].(string)
	routeTableId := req.ResourceProperties["RouteTableId"].(string)

	_, err := EC2(req).CreateRoute(&ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(destinationCidrBlock),
		NatGatewayId:         aws.String(req.ResourceProperties["NatGatewayId"].(string)),
		RouteTableId:         aws.String(routeTableId),
	})

	if err != nil {
		return "invalid", nil, err
	}

	return routeTableId + "/" + destinationCidrBlock, nil, nil
}

func EC2RouteUpdate(req Request) (string, map[string]string, error) {
	parts := strings.SplitN(req.PhysicalResourceId, "/", 2)

	destinationCidrBlock := req.ResourceProperties["DestinationCidrBlock"].(string)
	routeTableId := req.ResourceProperties["RouteTableId"].(string)

	if parts[0] == routeTableId && parts[1] == destinationCidrBlock {
		return req.PhysicalResourceId, nil, nil
	}

	_, err := EC2(req).DeleteRoute(&ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String(parts[1]),
		RouteTableId:         aws.String(parts[0]),
	})

	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	_, err = EC2(req).CreateRoute(&ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(destinationCidrBlock),
		NatGatewayId:         aws.String(req.ResourceProperties["NatGatewayId"].(string)),
		RouteTableId:         aws.String(routeTableId),
	})

	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	return routeTableId + "/" + destinationCidrBlock, nil, nil
}

func EC2RouteDelete(req Request) (string, map[string]string, error) {
	parts := strings.SplitN(req.PhysicalResourceId, "/", 2)

	_, err := EC2(req).DeleteRoute(&ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String(parts[1]),
		RouteTableId:         aws.String(parts[0]),
	})

	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "InvalidRoute.NotFound" {
			return req.PhysicalResourceId, nil, nil
		}
	}

	return req.PhysicalResourceId, nil, err
}
