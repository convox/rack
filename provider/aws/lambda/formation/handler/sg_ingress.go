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
	newSgStrs := strings.Split(req.ResourceProperties["SgIDs"].(string), ",")

	res, err := EC2(req).DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{sgId},
	})
	if err != nil {
		fmt.Println("1")
		return err
	}

	var ipPermissions []*ec2.IpPermission
	for _, group := range res.SecurityGroups {
		ipPermissions = group.IpPermissions

	}

	var prevIps []*ec2.IpRange
	var userDefinedIps []*ec2.IpRange
	var prevSgIDs []string
	var userDefinedSgIDs []string
	for i := range ipPermissions {
		if ipPermissions[i].IpProtocol != nil && *ipPermissions[i].IpProtocol == "tcp" &&
			ipPermissions[i].FromPort != nil && *ipPermissions[i].FromPort == 443 &&
			ipPermissions[i].ToPort != nil && *ipPermissions[i].ToPort == 443 {

			for _, ipRange := range ipPermissions[i].IpRanges {
				if ipRange.Description != nil && strings.Contains(*ipRange.Description, CONVOX_MANAGED) {
					prevIps = append(prevIps, &ec2.IpRange{
						CidrIp: ipRange.CidrIp,
					})
				} else {
					userDefinedIps = append(userDefinedIps, &ec2.IpRange{
						CidrIp: ipRange.CidrIp,
					})
				}
			}

			for _, idGroupPair := range ipPermissions[i].UserIdGroupPairs {
				if idGroupPair.Description != nil && strings.Contains(*idGroupPair.Description, CONVOX_MANAGED) && idGroupPair.GroupId != nil {
					prevSgIDs = append(prevSgIDs, *idGroupPair.GroupId)
				} else if idGroupPair.GroupId != nil {
					userDefinedSgIDs = append(userDefinedSgIDs, *idGroupPair.GroupId)
				}
			}
		}
	}

	prevIpMap := map[string]int{}
	for i := range prevIps {
		if prevIps[i].CidrIp != nil {
			prevIpMap[*prevIps[i].CidrIp] = 1
		}
	}

	for i := range userDefinedIps {
		if userDefinedIps[i].CidrIp != nil {
			prevIpMap[*userDefinedIps[i].CidrIp] = 2 // not to be touched
		}
	}

	var newIps []*ec2.IpRange
	for _, ipStr := range newIpStrs {
		cidr := strings.TrimSpace(ipStr)
		if cidr != "" {
			if _, has := prevIpMap[cidr]; !has {
				newIps = append(newIps, &ec2.IpRange{
					CidrIp:      aws.String(cidr),
					Description: aws.String(CONVOX_MANAGED),
				})
			}
			prevIpMap[cidr] = 3
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
			fmt.Println("3")
			return err
		}
	}

	var deleteIps []*ec2.IpRange
	for i := range prevIps {
		if prevIps[i].CidrIp != nil {
			if val, _ := prevIpMap[*prevIps[i].CidrIp]; val == 1 {
				deleteIps = append(deleteIps, prevIps[i])
			}
		}
	}
	if len(deleteIps) > 0 {
		_, err = EC2(req).RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
			GroupId: sgId,
			IpPermissions: []*ec2.IpPermission{
				{
					FromPort:   aws.Int64(443),
					ToPort:     aws.Int64(443),
					IpProtocol: aws.String("tcp"),
					IpRanges:   deleteIps,
				},
			},
		})
		if err != nil {
			fmt.Println("2")
			return err
		}
	}

	prevSgMap := map[string]int{}
	for _, id := range prevSgIDs {
		prevSgMap[id] = 1
	}

	for _, id := range userDefinedSgIDs {
		prevSgMap[id] = 2 // not be touched
	}

	var newGrpPairs []*ec2.UserIdGroupPair
	for _, id := range newSgStrs {
		id = strings.TrimSpace(id)
		if id != "" {
			if _, has := prevSgMap[id]; !has {
				newGrpPairs = append(newGrpPairs, &ec2.UserIdGroupPair{
					Description: aws.String(CONVOX_MANAGED),
					GroupId:     aws.String(id),
				})
			} else {
				prevSgMap[id] = 2
			}
		}
	}

	if len(newGrpPairs) > 0 {
		_, err = EC2(req).AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: sgId,
			IpPermissions: []*ec2.IpPermission{
				{
					FromPort:         aws.Int64(443),
					ToPort:           aws.Int64(443),
					IpProtocol:       aws.String("tcp"),
					UserIdGroupPairs: newGrpPairs,
				},
			},
		})
		if err != nil {
			fmt.Println("4")
			return err
		}
	}

	var deleteGrpPairs []*ec2.UserIdGroupPair
	for id, val := range prevSgMap {
		if val == 1 {
			deleteGrpPairs = append(deleteGrpPairs, &ec2.UserIdGroupPair{
				GroupId: aws.String(id),
			})
		}
	}

	if len(deleteGrpPairs) > 0 {
		_, err = EC2(req).RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
			GroupId: sgId,
			IpPermissions: []*ec2.IpPermission{
				{
					FromPort:         aws.Int64(443),
					ToPort:           aws.Int64(443),
					IpProtocol:       aws.String("tcp"),
					UserIdGroupPairs: deleteGrpPairs,
				},
			},
		})
		if err != nil {
			fmt.Println("5")
			return err
		}
	}
	return nil
}
