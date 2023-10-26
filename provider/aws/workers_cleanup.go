package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/logger"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
)

const CONVOX_INSTANCE_MANAGED = "CONVOX_INSTANCE"

func (p *Provider) workerCleanup() {
	log := logger.New("ns=workers.cleanup")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for range time.Tick(1 * time.Hour) {
		p.cleanupBuilds(log)
	}
}

func (p *Provider) workerSyncInstanceIPs() {
	log := logger.New("ns=workers.syncInstanceips")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for range time.Tick(30 * time.Minute) {
		err := p.SyncInstancesIpInSecurityGroup()
		log.Error(err)
	}
}

func (p *Provider) cleanupBuilds(log *logger.Logger) error {
	as, err := p.AppList()
	if err != nil {
		return log.Error(err)
	}

	for _, a := range as {
		log = log.Replace("app", a.Name)

		log = log.At("images")
		if count, err := p.cleanupAppImages(a); err != nil {
			log.Error(err)
		} else {
			log.Logf("expired=%d", count)
		}

		log = log.At("builds")
		if count, err := p.cleanupAppBuilds(a); err != nil {
			log.Error(err)
		} else {
			log.Logf("expired=%d", count)
		}
	}

	return nil
}

func (p *Provider) cleanupAppBuilds(a structs.App) (int, error) {
	active, err := p.activeBuild(a)
	if err != nil {
		return 0, err
	}

	removed := 0

	for {
		bs, err := p.BuildList(a.Name, structs.BuildListOptions{Limit: options.Int(1000)})
		if err != nil {
			return 0, err
		}

		if len(bs) <= maxBuilds {
			break
		}

		remove := []string{}

		for _, b := range bs[maxBuilds:] {
			if b.Id != active {
				remove = append(remove, b.Id)
			}
		}

		for _, rc := range chunk(remove, 25) {
			req := &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{p.DynamoBuilds: {}},
			}

			for _, id := range rc {
				req.RequestItems[p.DynamoBuilds] = append(req.RequestItems[p.DynamoBuilds], &dynamodb.WriteRequest{
					DeleteRequest: &dynamodb.DeleteRequest{
						Key: map[string]*dynamodb.AttributeValue{
							"id": {S: aws.String(id)},
						},
					},
				})
			}

			if _, err := p.dynamodb().BatchWriteItem(req); err != nil {
				return 0, err
			}

			removed += len(rc)
		}
	}

	return removed, nil
}

func (p *Provider) cleanupAppImages(a structs.App) (int, error) {
	active, err := p.activeBuild(a)
	if err != nil {
		return 0, err
	}

	bs, err := p.BuildList(a.Name, structs.BuildListOptions{Limit: options.Int(maxBuilds)})
	if err != nil {
		return 0, err
	}

	if len(bs) < maxBuilds {
		return 0, nil
	}

	bh := map[string]bool{}

	for _, b := range bs {
		bh[b.Id] = true
	}

	repo, err := p.appRepositoryName(a)
	if err != nil {
		return 0, err
	}

	tags, err := p.repoTags(repo)
	if err != nil {
		return 0, err
	}

	remove := []string{}

	for _, tag := range tags {
		parts := strings.SplitN(tag, ".", 2)
		if len(parts) < 2 || !strings.HasPrefix(parts[1], "B") {
			continue
		}

		if _, ok := bh[parts[1]]; !ok && parts[1] != active {
			remove = append(remove, tag)
		}
	}

	if len(remove) == 0 {
		return 0, nil
	}

	for _, rc := range chunk(remove, 100) {
		req := &ecr.BatchDeleteImageInput{
			RepositoryName: aws.String(repo),
		}

		for _, tag := range rc {
			req.ImageIds = append(req.ImageIds, &ecr.ImageIdentifier{
				ImageTag: aws.String(tag),
			})
		}

		if _, err := p.ecr().BatchDeleteImage(req); err != nil {
			return 0, err
		}
	}

	return len(remove), nil
}

func (p *Provider) activeBuild(a structs.App) (string, error) {
	if a.Release == "" {
		return "", nil
	}

	r, err := p.ReleaseGet(a.Name, a.Release)
	if err != nil {
		return "", err
	}

	return r.Build, nil
}

func (p *Provider) repoTags(repo string) ([]string, error) {
	tags := map[string]bool{}

	err := p.ecr().ListImagesPages(&ecr.ListImagesInput{RepositoryName: aws.String(repo)}, func(page *ecr.ListImagesOutput, last bool) bool {
		for _, i := range page.ImageIds {
			if i.ImageTag != nil && *i.ImageTag != "" {
				tags[*i.ImageTag] = true
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	ts := []string{}

	for t := range tags {
		ts = append(ts, t)
	}

	return ts, nil
}

func (p *Provider) appRepositoryName(a structs.App) (string, error) {
	switch a.Generation {
	case "1":
		return a.Outputs["RegistryRepository"], nil
	case "2":
		return p.appResource(a.Name, "Registry")
	}

	return "", fmt.Errorf("unknown generation: %s", a.Generation)
}

func chunk(ss []string, count int) [][]string {
	chunks := [][]string{}

	for {
		if len(ss) <= count {
			return append(chunks, ss)
		}

		chunks = append(chunks, ss[0:count])
		ss = ss[count:]
	}
}

func (p *Provider) SyncInstancesIpInSecurityGroup() error {
	log := logger.New("ns=workers.syncInstancesIpInSecurityGroup")
	sgId := p.ApiBalancerSecurity

	log.Logf("cleanup and sync instances ip in security group")
	ipHash := map[string]bool{}
	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Rack"), Values: []*string{aws.String(p.Rack)}},
		},
	}

	err := p.ec2().DescribeInstancesPages(req, func(res *ec2.DescribeInstancesOutput, last bool) bool {
		for _, r := range res.Reservations {
			for _, i := range r.Instances {
				if i.PublicIpAddress != nil && *i.PublicIpAddress != "" {
					ipHash[*i.PublicIpAddress+"/32"] = true
				}
			}
		}
		return true
	})
	if err != nil {
		return err
	}

	res, err := p.ec2().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{&sgId},
	})
	if err != nil {
		return err
	}

	var ipPermissions []*ec2.IpPermission
	for _, group := range res.SecurityGroups {
		ipPermissions = group.IpPermissions
	}

	var prevIps []*ec2.IpRange
	var userDefinedIps []*ec2.IpRange
	for i := range ipPermissions {
		if ipPermissions[i].IpProtocol != nil && *ipPermissions[i].IpProtocol == "tcp" &&
			ipPermissions[i].FromPort != nil && *ipPermissions[i].FromPort == 443 &&
			ipPermissions[i].ToPort != nil && *ipPermissions[i].ToPort == 443 {

			for _, ipRange := range ipPermissions[i].IpRanges {
				if ipRange.Description != nil && strings.Contains(*ipRange.Description, CONVOX_INSTANCE_MANAGED) {
					prevIps = append(prevIps, &ec2.IpRange{
						CidrIp: ipRange.CidrIp,
					})
				} else {
					userDefinedIps = append(userDefinedIps, &ec2.IpRange{
						CidrIp: ipRange.CidrIp,
					})
				}
			}
		}
	}

	if p.Private || !p.WhiteListSpecified {
		log.Logf("clean up instances ips if they exists")
		if len(prevIps) > 0 {
			_, err = p.ec2().RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
				GroupId: &sgId,
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
		return nil
	}

	var deleteIps []*ec2.IpRange
	for i := range prevIps {
		if !ipHash[*prevIps[i].CidrIp] {
			deleteIps = append(deleteIps, prevIps[i])
		} else {
			ipHash[*prevIps[i].CidrIp] = false
		}
	}

	for i := range userDefinedIps {
		if ipHash[*userDefinedIps[i].CidrIp] {
			ipHash[*prevIps[i].CidrIp] = false
		}
	}

	var newIps []*ec2.IpRange
	for k, v := range ipHash {
		if v {
			newIps = append(newIps, &ec2.IpRange{
				CidrIp:      aws.String(k),
				Description: aws.String(CONVOX_INSTANCE_MANAGED),
			})
		}
	}

	if len(newIps) > 0 {
		_, err = p.ec2().AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: &sgId,
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

	if len(deleteIps) > 0 {
		_, err = p.ec2().RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
			GroupId: &sgId,
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
			return err
		}
	}

	return nil
}
