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

const (
	metricPrecision = 5
)

var (
	metricShifter = math.Pow(10, metricPrecision)
)

func (p *Provider) cloudwatchMetrics(mdqs []metricDataQuerier, opts structs.MetricsOptions) (structs.Metrics, error) {
	period := ci(opts.Period, 3600)

	req := &cloudwatch.GetMetricDataInput{
		EndTime:           aws.Time(ct(opts.End, time.Now())),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{},
		ScanBy:            aws.String("TimestampAscending"),
		StartTime:         aws.Time(ct(opts.Start, time.Now().Add(-24*time.Hour))),
	}

	for i, mdq := range mdqs {
		req.MetricDataQueries = append(req.MetricDataQueries, mdq.MetricDataQueries(period, fmt.Sprintf("%d", i))...)
	}

	res, err := p.CloudWatch.GetMetricData(req)
	if err != nil {
		return nil, err
	}

	msh := map[string]map[time.Time]structs.MetricValue{}

	for _, dr := range res.MetricDataResults {
		if dr.Label == nil {
			continue
		}

		parts := strings.SplitN(*dr.Label, "/", 2)

		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		stat := parts[1]

		mvsh, ok := msh[name]
		if !ok {
			mvsh = map[time.Time]structs.MetricValue{}
		}

		for i, ts := range dr.Timestamps {
			if ts == nil {
				continue
			}

			if dr.Values[i] == nil {
				continue
			}

			v := math.Floor(*dr.Values[i]*metricShifter) / metricShifter

			mv, ok := mvsh[*ts]
			if !ok {
				mv = structs.MetricValue{Time: *ts}
			}

			switch stat {
			case "Average":
				mv.Average += v
			case "Minimum":
				mv.Minimum += v
			case "Maximum":
				mv.Maximum += v
			case "p90":
				mv.P90 += v
			case "p95":
				mv.P95 += v
			case "p99":
				mv.P99 += v
			case "SampleCount":
				mv.Count += v / (float64(period) / 60)
			case "Sum":
				mv.Sum += v
			}

			mvsh[*ts] = mv
		}

		msh[name] = mvsh
	}

	ms := structs.Metrics{}

	for name, mvsh := range msh {
		m := structs.Metric{Name: name}

		mvs := structs.MetricValues{}

		for _, mv := range mvsh {
			mvs = append(mvs, mv)
		}

		sort.Slice(mvs, func(i, j int) bool { return mvs[i].Time.Before(mvs[j].Time) })

		m.Values = mvs

		ms = append(ms, m)
	}

	sort.Slice(ms, func(i, j int) bool { return ms[i].Name < ms[j].Name })

	return ms, nil
}

func (p *Provider) appMetricQueries(app string) ([]metricDataQuerier, error) {
	rs, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.rackStack(app)),
	})
	if err != nil {
		return nil, err
	}

	sr, err := p.describeStack(p.Rack)
	if err != nil {
		return nil, err
	}

	ros := stackOutputs(sr)

	mdqs := []metricDataQuerier{}

	for _, r := range rs.StackResources {
		if r.ResourceType == nil || r.LogicalResourceId == nil {
			continue
		}

		if *r.ResourceType != "AWS::CloudFormation::Stack" || !strings.HasPrefix(*r.LogicalResourceId, "Service") {
			continue
		}

		s, err := p.describeStack(*r.PhysicalResourceId)
		if err != nil {
			return nil, err
		}

		sos := stackOutputs(s)

		if sv := sos["Service"]; sv != "" {
			svp := strings.Split(sv, "/")
			svn := svp[len(svp)-1]

			mdqs = append(mdqs, metricExpressions{
				"process:running",
				[]metricStatistics{
					metricStatistics{"running_count", "AWS/ECS", "CPUUtilization", map[string]string{"ClusterName": p.Cluster, "ServiceName": svn}, []string{"SampleCount"}},
				},
				[]metricExpression{
					metricExpression{"SampleCount", "FILL(running_count_SampleCount_##,0)"},
				},
			})
		}

		if tg := sos["TargetGroup"]; tg != "" {
			tgp := strings.Split(tg, ":")
			tgn := tgp[len(tgp)-1]

			if rn := ros["RouterName"]; rn != "" {
				ns := "AWS/ApplicationELB"
				dim := map[string]string{"LoadBalancer": rn, "TargetGroup": tgn}

				mdqs = append(mdqs, metricStatistics{"process:healthy", ns, "HealthyHostCount", dim, []string{"Average", "Minimum", "Maximum"}})
				mdqs = append(mdqs, metricStatistics{"process:unhealthy", ns, "UnHealthyHostCount", dim, []string{"Average", "Minimum", "Maximum"}})

				mdqs = append(mdqs, metricExpressions{
					"service:requests",
					[]metricStatistics{
						metricStatistics{"target_requests", ns, "RequestCountPerTarget", dim, []string{"Sum"}},
						metricStatistics{"target_count", ns, "HealthyHostCount", dim, []string{"Average"}},
					},
					[]metricExpression{
						metricExpression{"Sum", "FILL(target_requests_Sum_##,0)*FILL(target_count_Average_##,0)"},
					},
				})
			}
		}
	}

	return mdqs, nil
}

func (p *Provider) serviceMetricQueries(app, service string) ([]metricDataQuerier, error) {
	sr, err := p.describeStack(p.Rack)
	if err != nil {
		return nil, err
	}

	ros := stackOutputs(sr)

	r, err := p.describeStackResource(&cloudformation.DescribeStackResourceInput{
		LogicalResourceId: aws.String(fmt.Sprintf("Service%s", upperName(service))),
		StackName:         aws.String(p.rackStack(app)),
	})
	if err != nil {
		return nil, err
	}

	s, err := p.describeStack(*r.StackResourceDetail.PhysicalResourceId)
	if err != nil {
		return nil, err
	}

	sos := stackOutputs(s)

	mdqs := []metricDataQuerier{}

	if sv := sos["Service"]; sv != "" {
		svp := strings.Split(sv, "/")
		svn := svp[len(svp)-1]

		mdqs = append(mdqs, metricExpressions{
			"process:running",
			[]metricStatistics{
				metricStatistics{"running_count", "AWS/ECS", "CPUUtilization", map[string]string{"ClusterName": p.Cluster, "ServiceName": svn}, []string{"SampleCount"}},
			},
			[]metricExpression{
				metricExpression{"SampleCount", "FILL(running_count_SampleCount_##,0)"},
			},
		})
	}

	if rn := ros["RouterName"]; rn != "" {
		ns := "AWS/ApplicationELB"

		if tg := sos["TargetGroup"]; tg != "" {
			tgp := strings.Split(tg, ":")
			tgn := tgp[len(tgp)-1]

			dim := map[string]string{"LoadBalancer": rn, "TargetGroup": tgn}

			mdqs = append(mdqs, metricStatistics{"process:healthy", ns, "HealthyHostCount", dim, []string{"Average", "Minimum", "Maximum"}})
			mdqs = append(mdqs, metricStatistics{"process:unhealthy", ns, "UnHealthyHostCount", dim, []string{"Average", "Minimum", "Maximum"}})
			mdqs = append(mdqs, metricStatistics{"service:requests:2xx", ns, "HTTPCode_Target_2XX_Count", dim, []string{"Sum"}})
			mdqs = append(mdqs, metricStatistics{"service:requests:3xx", ns, "HTTPCode_Target_3XX_Count", dim, []string{"Sum"}})
			mdqs = append(mdqs, metricStatistics{"service:requests:4xx", ns, "HTTPCode_Target_4XX_Count", dim, []string{"Sum"}})
			mdqs = append(mdqs, metricStatistics{"service:requests:5xx", ns, "HTTPCode_Target_5XX_Count", dim, []string{"Sum"}})

			mdqs = append(mdqs, metricExpressions{
				"service:response:time",
				[]metricStatistics{
					metricStatistics{"response_time", ns, "TargetResponseTime", dim, []string{"Average", "Minimum", "Maximum", "p90", "p95", "p99"}},
				},
				[]metricExpression{
					metricExpression{"Average", "FILL(response_time_Average_##,0)"},
					metricExpression{"Minimum", "FILL(response_time_Minimum_##,0)"},
					metricExpression{"Maximum", "FILL(response_time_Maximum_##,0)"},
					metricExpression{"p90", "FILL(response_time_p90_##,0)"},
					metricExpression{"p95", "FILL(response_time_p95_##,0)"},
					metricExpression{"p99", "FILL(response_time_Maximum_##,0)"},
				},
			})

			mdqs = append(mdqs, metricExpressions{
				"service:requests",
				[]metricStatistics{
					metricStatistics{"target_requests", ns, "RequestCountPerTarget", dim, []string{"Sum"}},
					metricStatistics{"target_count", ns, "HealthyHostCount", dim, []string{"Average"}},
				},
				[]metricExpression{
					metricExpression{"Sum", "FILL(target_requests_Sum_##,0)*FILL(target_count_Average_##,0)"},
				},
			})

			// mdqs = append(mdqs, metricStatistics{"service:response:time", ns, "TargetResponseTime", dim, []string{"Minimum", "Maximum", "p90", "p95", "p99"}})
		}
	}

	return mdqs, nil
}

func (p *Provider) systemMetricQueries() []metricDataQuerier {
	mdqs := []metricDataQuerier{
		metricStatistics{"cluster:cpu:reservation", "AWS/ECS", "CPUReservation", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		metricStatistics{"cluster:cpu:utilization", "AWS/ECS", "CPUUtilization", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		metricStatistics{"cluster:mem:reservation", "AWS/ECS", "MemoryReservation", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		metricStatistics{"cluster:mem:utilization", "AWS/ECS", "MemoryUtilization", map[string]string{"ClusterName": p.Cluster}, []string{"Average", "Minimum", "Maximum"}},
		metricStatistics{"instances:standard:cpu", "AWS/EC2", "CPUUtilization", map[string]string{"AutoScalingGroupName": p.AsgStandard}, []string{"Average", "Minimum", "Maximum"}},
	}

	if p.AsgSpot != "" {
		mdqs = append(mdqs, metricStatistics{"instances:spot:cpu", "AWS/EC2", "CPUUtilization", map[string]string{"AutoScalingGroupName": p.AsgSpot}, []string{"Average", "Minimum", "Maximum"}})
	}

	return mdqs
}

func (p *Provider) servicesMetricQueries(names []string) []metricDataQuerier {
	mdqs := []metricDataQuerier{}

	for _, name := range names {
		mdqs = append(mdqs, metricStatistics{
			Name:      serviceMetricsKey("cpu", name),
			Namespace: "AWS/ECS",
			Metric:    "CPUUtilization",
			Dimensions: map[string]string{
				"ClusterName": p.Cluster,
				"ServiceName": name,
			},
			Statistics: []string{"Average"},
		}, metricStatistics{
			Name:      serviceMetricsKey("mem", name),
			Namespace: "AWS/ECS",
			Metric:    "MemoryUtilization",
			Dimensions: map[string]string{
				"ClusterName": p.Cluster,
				"ServiceName": name,
			},
			Statistics: []string{"Average"},
		})
	}

	return mdqs
}

type metricDataQuerier interface {
	MetricDataQueries(period int64, suffix string) []*cloudwatch.MetricDataQuery
}

type metricExpressions struct {
	Name        string
	Statistics  []metricStatistics
	Expressions []metricExpression
}

type metricExpression struct {
	Statistic  string
	Expression string
}

type metricStatistics struct {
	Name       string
	Namespace  string
	Metric     string
	Dimensions map[string]string
	Statistics []string
}

func (me metricExpressions) MetricDataQueries(period int64, suffix string) []*cloudwatch.MetricDataQuery {
	qs := []*cloudwatch.MetricDataQuery{}

	for _, ms := range me.Statistics {
		qs = append(qs, ms.MetricDataQueries(period, suffix)...)
	}

	for _, e := range me.Expressions {
		q := &cloudwatch.MetricDataQuery{
			Id:         aws.String(fmt.Sprintf("%s_%s_%s", strings.ReplaceAll(me.Name, ":", "_"), e.Statistic, suffix)),
			Label:      aws.String(fmt.Sprintf("%s/%s", me.Name, e.Statistic)),
			Expression: aws.String(strings.ReplaceAll(e.Expression, "##", suffix)),
		}

		qs = append(qs, q)
	}

	return qs
}

func (ms metricStatistics) MetricDataQueries(period int64, suffix string) []*cloudwatch.MetricDataQuery {
	qs := []*cloudwatch.MetricDataQuery{}

	dim := []*cloudwatch.Dimension{}
	for k, v := range ms.Dimensions {
		dim = append(dim, &cloudwatch.Dimension{
			Name:  options.String(k),
			Value: options.String(v),
		})
	}

	stats := []*string{}
	for _, s := range ms.Statistics {
		stats = append(stats, aws.String(s))
	}

	for _, s := range ms.Statistics {
		q := &cloudwatch.MetricDataQuery{
			Id:    aws.String(fmt.Sprintf("%s_%s_%s", strings.ReplaceAll(ms.Name, ":", "_"), s, suffix)),
			Label: aws.String(fmt.Sprintf("%s/%s", ms.Name, s)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Dimensions: dim,
					MetricName: aws.String(ms.Metric),
					Namespace:  aws.String(ms.Namespace),
				},
				Period: aws.Int64(period),
				Stat:   aws.String(s),
			},
		}

		qs = append(qs, q)
	}

	return qs
}
