package aws

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
)

type metricDefinition struct {
	Name       string
	Namespace  string
	Metric     string
	Dimensions map[string]string
	Statistics []string
}

func (p *Provider) cloudwatchMetric(md metricDefinition, opts structs.MetricsOptions) (*structs.Metric, error) {
	dim := []*cloudwatch.Dimension{}

	for k, v := range md.Dimensions {
		dim = append(dim, &cloudwatch.Dimension{
			Name:  options.String(k),
			Value: options.String(v),
		})
	}

	stats := []*string{}

	for _, s := range md.Statistics {
		stats = append(stats, aws.String(s))
	}

	req := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: dim,
		EndTime:    aws.Time(ct(opts.End, time.Now())),
		MetricName: aws.String(md.Metric),
		Namespace:  aws.String(md.Namespace),
		Period:     aws.Int64(ci(opts.Period, 3600)),
		StartTime:  aws.Time(ct(opts.Start, time.Now().Add(-24*time.Hour))),
		Statistics: stats,
	}

	res, err := p.CloudWatch.GetMetricStatistics(req)
	if err != nil {
		return nil, err
	}

	mvs := structs.MetricValues{}

	for _, d := range res.Datapoints {
		mv := structs.MetricValue{
			Time: *d.Timestamp,
		}

		if d.Average != nil {
			mv.Average = math.Floor(*d.Average*100) / 100
		}

		if d.Minimum != nil {
			mv.Minimum = math.Floor(*d.Minimum*100) / 100
		}

		if d.Maximum != nil {
			mv.Maximum = math.Floor(*d.Maximum*100) / 100
		}

		if d.SampleCount != nil {
			mv.Count = math.Floor((*d.SampleCount/(float64(*req.Period)/60))*100) / 100
		}

		mvs = append(mvs, mv)
	}

	sort.Slice(mvs, func(i, j int) bool { return mvs[i].Time.Before(mvs[j].Time) })

	m := &structs.Metric{
		Name:   md.Name,
		Values: mvs,
	}

	return m, nil
}

func (p *Provider) appMetricDefinitions(app string) ([]metricDefinition, error) {
	rs, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.rackStack(app)),
	})
	if err != nil {
		return nil, err
	}

	mds := []metricDefinition{}

	for _, r := range rs.StackResources {
		if r.ResourceType != nil && r.LogicalResourceId != nil {
			if *r.ResourceType == "AWS::CloudFormation::Stack" && strings.HasPrefix(*r.LogicalResourceId, "Service") {
				s, err := p.describeStack(*r.PhysicalResourceId)
				if err != nil {
					return nil, err
				}

				sos := stackOutputs(s)

				if sv := sos["Service"]; sv != "" {
					svp := strings.Split(sv, "/")
					svn := svp[len(svp)-1]
					fmt.Printf("svn = %+v\n", svn)

					mds = append(mds, metricDefinition{"process:running", "AWS/ECS", "CPUUtilization", map[string]string{"ClusterName": p.Cluster, "ServiceName": svn}, []string{"SampleCount"}})
				}

				fmt.Printf("sos = %+v\n", sos)

				// if tg := sos["TargetGroup"]; tg != "" {
				//   fmt.Printf("tg = %+v\n", tg)
				// }
			}
		}
	}

	fmt.Printf("mds = %+v\n", mds)

	// fmt.Printf("rs = %+v\n", rs)

	return mds, nil
}

func (p *Provider) systemMetricDefinitions() []metricDefinition {
	mds := []metricDefinition{
		{"cluster:cpu:reservation", "AWS/ECS", "CPUReservation", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		{"cluster:cpu:utilization", "AWS/ECS", "CPUUtilization", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		{"cluster:mem:reservation", "AWS/ECS", "MemoryReservation", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		{"cluster:mem:utilization", "AWS/ECS", "MemoryUtilization", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		{"instances:standard:cpu", "AWS/EC2", "CPUUtilization", map[string]string{"AutoScalingGroupName": p.AsgStandard}, []string{"Average", "Minimum", "Maximum"}},
	}

	if p.AsgSpot != "" {
		mds = append(mds, metricDefinition{"instances:spot:cpu", "AWS/EC2", "CPUUtilization", map[string]string{"AutoScalingGroupName": p.AsgSpot}, []string{"Average", "Minimum", "Maximum"}})
	}

	return mds
}
