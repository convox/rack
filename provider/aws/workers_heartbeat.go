package aws

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/logger"
	"github.com/convox/rack/pkg/helpers"
)

func (p *Provider) workerHeartbeat() {
	helpers.Tick(1*time.Hour, p.heartbeat)
}

func (p *Provider) heartbeat() {
	var log = logger.New("ns=workers.heartbeat")

	s, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	as, err := p.AppList()
	if err != nil {
		log.Error(err)
		return
	}

	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Rack"), Values: []*string{aws.String(p.Rack)}},
			{Name: aws.String("tag:aws:cloudformation:logical-id"), Values: []*string{aws.String("Instances"), aws.String("SpotInstances")}},
			{Name: aws.String("instance-state-name"), Values: []*string{aws.String("pending"), aws.String("running"), aws.String("shutting-down"), aws.String("stopping")}},
		},
	}

	onDemandCnt := 0
	spotCnt := 0

	err = p.ec2().DescribeInstancesPages(req, func(res *ec2.DescribeInstancesOutput, last bool) bool {
		for _, r := range res.Reservations {
			for _, i := range r.Instances {
				if i.InstanceLifecycle != nil && strings.ToLower(*i.InstanceLifecycle) == "spot" {
					spotCnt++
				} else {
					onDemandCnt++
				}
			}
		}
		return true
	})
	if err != nil {
		log.Error(err)
		return
	}

	ms := map[string]interface{}{
		"id":                       coalesces(p.ClientId, p.StackId),
		"app_count":                len(as),
		"instance_count":           s.Count,
		"instance_type":            s.Type,
		"provider":                 "aws",
		"rack_id":                  p.StackId,
		"region":                   p.Region,
		"version":                  s.Version,
		"on_demand_instance_count": onDemandCnt,
		"spot_instance_count":      spotCnt,
	}

	telemetryOn := s.Parameters["Telemetry"] == "true"

	if telemetryOn {
		params := p.RackParamsToSync(s.Version, s.Parameters)
		ms["rack_params"] = params
	}

	if err := p.Metrics.Post("heartbeat", ms); err != nil {
		log.Error(err)
		return
	}
}
