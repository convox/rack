package aws

import (
	"math"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
)

type metricDefinition struct {
	Name       string
	Namespace  string
	Metric     string
	Dimensions map[string]string
}

func (p *Provider) cloudwatchMetric(md metricDefinition, opts structs.MetricsOptions) (*structs.Metric, error) {
	dim := []*cloudwatch.Dimension{}

	for k, v := range md.Dimensions {
		dim = append(dim, &cloudwatch.Dimension{
			Name:  options.String(k),
			Value: options.String(v),
		})
	}

	req := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: dim,
		EndTime:    aws.Time(ct(opts.End, time.Now())),
		MetricName: aws.String(md.Metric),
		Namespace:  aws.String(md.Namespace),
		Period:     aws.Int64(ci(opts.Period, 3600)),
		StartTime:  aws.Time(ct(opts.Start, time.Now().Add(-24*time.Hour))),
		Statistics: []*string{aws.String("Average"), aws.String("Minimum"), aws.String("Maximum")},
	}

	res, err := p.CloudWatch.GetMetricStatistics(req)
	if err != nil {
		return nil, err
	}

	mvs := structs.MetricValues{}

	for _, d := range res.Datapoints {
		mvs = append(mvs, structs.MetricValue{
			Time:    *d.Timestamp,
			Minimum: math.Floor(*d.Minimum*100) / 100,
			Average: math.Floor(*d.Average*100) / 100,
			Maximum: math.Floor(*d.Maximum*100) / 100,
		})
	}

	sort.Slice(mvs, func(i, j int) bool { return mvs[i].Time.Before(mvs[j].Time) })

	m := &structs.Metric{
		Name:   md.Name,
		Values: mvs,
	}

	return m, nil
}

func (p *Provider) systemMetricDefinitions() []metricDefinition {
	mds := []metricDefinition{
		{"cluster:cpu:reservation", "AWS/ECS", "CPUReservation", map[string]string{"ClusterName": p.Cluster}},
		{"cluster:cpu:utilization", "AWS/ECS", "CPUUtilization", map[string]string{"ClusterName": p.Cluster}},
		{"cluster:mem:reservation", "AWS/ECS", "MemoryReservation", map[string]string{"ClusterName": p.Cluster}},
		{"cluster:mem:utilization", "AWS/ECS", "MemoryUtilization", map[string]string{"ClusterName": p.Cluster}},
		{"instances:standard:cpu", "AWS/EC2", "CPUUtilization", map[string]string{"AutoScalingGroupName": p.AsgStandard}},
	}

	if p.AsgSpot != "" {
		mds = append(mds, metricDefinition{"instances:spot:cpu", "AWS/EC2", "CPUUtilization", map[string]string{"AutoScalingGroupName": p.AsgSpot}})
	}

	return mds
}
