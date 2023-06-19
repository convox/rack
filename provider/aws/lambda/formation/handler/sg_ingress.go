package handler

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const CONVOX_MANAGED = "convox managed"

func HandleSGIngress(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING SG ingresses")
		fmt.Printf("req %+v\n", req)
		return SGIngressCreate(req)
	case "Update":
		fmt.Println("UPDATING SG ingresses")
		fmt.Printf("req %+v\n", req)
		return SGIngressUpdate(req)
	case "Delete":
		fmt.Println("no need to delete, since sg will be deleted")
		fmt.Printf("req %+v\n", req)
		return req.PhysicalResourceId, nil, nil
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func SGIngressCreate(req Request) (string, map[string]string, error) {
	err := sgIngressApply(req)
	if err != nil {
		return "invalid", nil, err
	}

	return fmt.Sprintf("ingress-%s", req.ResourceProperties["SecurityGroupID"].(string)), nil, nil
}

func SGIngressUpdate(req Request) (string, map[string]string, error) {
	err := sgIngressApply(req)
	if err != nil {
		return "invalid", nil, err
	}

	return req.PhysicalResourceId, nil, nil
}

func sgIngressApply(req Request) error {
	sgId := aws.String(req.ResourceProperties["SecurityGroupID"].(string))
	newIpStrs := strings.Split(req.ResourceProperties["Ips"].(string), ",")

	res, err := EC2(req).DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{sgId},
	})
	if err != nil {
		return err
	}

	var ipPermissions []*ec2.IpPermission
	for _, group := range res.SecurityGroups {
		ipPermissions = group.IpPermissions
	}

	var prevIps []*ec2.IpRange
	for i := range ipPermissions {
		if ipPermissions[i].IpProtocol != nil && *ipPermissions[i].IpProtocol == "tcp" &&
			ipPermissions[i].FromPort != nil && *ipPermissions[i].FromPort == 443 &&
			ipPermissions[i].ToPort != nil && *ipPermissions[i].ToPort == 443 {
			for _, ipRange := range ipPermissions[i].IpRanges {
				if ipRange.Description != nil && strings.Contains(*ipRange.Description, CONVOX_MANAGED) {
					prevIps = append(prevIps, &ec2.IpRange{
						CidrIp: ipRange.CidrIp,
					})
				}
			}
		}
	}

	if len(prevIps) > 0 {
		_, err = EC2(req).RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
			GroupId: sgId,
			IpPermissions: []*ec2.IpPermission{
				{
					FromPort:   aws.Int64(443),
					ToPort:     aws.Int64(443),
					IpProtocol: aws.String("tcp"),
					IpRanges:   prevIps,
				},
			},
		})
		if err != nil {
			return err
		}
	}

	var newIps []*ec2.IpRange
	for _, ipStr := range newIpStrs {
		cidr := strings.TrimSpace(ipStr)
		if cidr != "" {
			newIps = append(newIps, &ec2.IpRange{
				CidrIp:      aws.String(cidr),
				Description: aws.String(CONVOX_MANAGED),
			})
		}
	}

	if len(newIps) > 0 {
		_, err = EC2(req).AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: sgId,
			IpPermissions: []*ec2.IpPermission{
				{
					FromPort:   aws.Int64(443),
					ToPort:     aws.Int64(443),
					IpProtocol: aws.String("tcp"),
					IpRanges:   newIps,
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
