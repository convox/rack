package models

import (
	"fmt"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatch"
)

type Metrics struct {
	Cpu    float64
	Memory float64
	Disk   float64
}

func AppMetrics(app string) (*Metrics, error) {
	dimensions := []*cloudwatch.Dimension{
		&cloudwatch.Dimension{Name: aws.String("App"), Value: aws.String(app)},
		&cloudwatch.Dimension{Name: aws.String("Process"), Value: aws.String("<all>")},
		&cloudwatch.Dimension{Name: aws.String("InstanceId"), Value: aws.String("<all>")},
	}

	cpu, err := getMetric("CpuUtilization", dimensions, 1, 1)

	if err != nil {
		return nil, err
	}

	memory, err := getMetric("MemoryUtilization", dimensions, 1, 1)

	if err != nil {
		return nil, err
	}

	disk, err := getMetric("DiskUtilization", dimensions, 1, 1)

	if err != nil {
		return nil, err
	}

	return &Metrics{Cpu: getLastAverage(cpu), Memory: getLastAverage(memory), Disk: getLastAverage(disk)}, nil
}

func ProcessMetrics(app, process string) (*Metrics, error) {
	// dimensions := []*cloudwatch.Dimension{
	//   &cloudwatch.Dimension{Name: aws.String("App"), Value: aws.String(app)},
	//   &cloudwatch.Dimension{Name: aws.String("Process"), Value: aws.String(process)},
	//   &cloudwatch.Dimension{Name: aws.String("InstanceId"), Value: aws.String("<all>")},
	// }

	// cpu, err := getMetric("CpuUtilization", dimensions, 1, 1)

	// if err != nil {
	//   return nil, err
	// }

	// memory, err := getMetric("MemoryUtilization", dimensions, 1, 1)

	// if err != nil {
	//   return nil, err
	// }

	// disk, err := getMetric("DiskUtilization", dimensions, 1, 1)

	// if err != nil {
	//   return nil, err
	// }

	// return &Metrics{Cpu: getLastAverage(cpu), Memory: getLastAverage(memory), Disk: getLastAverage(disk)}, nil
	return &Metrics{Cpu: 0, Memory: 0, Disk: 0}, nil
}

func InstanceMetrics(app, process, instance string) (*Metrics, error) {
	dimensions := []*cloudwatch.Dimension{
		&cloudwatch.Dimension{Name: aws.String("App"), Value: aws.String(app)},
		&cloudwatch.Dimension{Name: aws.String("Process"), Value: aws.String(process)},
		&cloudwatch.Dimension{Name: aws.String("InstanceId"), Value: aws.String(instance)},
	}

	cpu, err := getMetric("CpuUtilization", dimensions, 1, 1)

	if err != nil {
		return nil, err
	}

	memory, err := getMetric("MemoryUtilization", dimensions, 1, 1)

	if err != nil {
		return nil, err
	}

	disk, err := getMetric("DiskUtilization", dimensions, 1, 1)

	if err != nil {
		return nil, err
	}

	return &Metrics{Cpu: getLastAverage(cpu), Memory: getLastAverage(memory), Disk: getLastAverage(disk)}, nil
}

func getMetric(metric string, dimensions []*cloudwatch.Dimension, span, precision int64) ([]*cloudwatch.Datapoint, error) {
	req := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: dimensions,
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String(metric),
		Namespace:  aws.String("Convox"),
		Period:     aws.Int64(precision * 60),
		StartTime:  aws.Time(time.Now().Add(time.Duration(-1*span) * time.Minute)),
		Statistics: []*string{aws.String("Average")},
	}

	res, err := CloudWatch().GetMetricStatistics(req)

	if err != nil {
		// TODO log error
		fmt.Printf("error fetching metrics: %s\n", err)
		return []*cloudwatch.Datapoint{}, nil
	}

	return res.Datapoints, nil
}

func getLastAverage(data []*cloudwatch.Datapoint) float64 {
	if len(data) < 1 {
		return 0
	} else {
		return *data[0].Average
	}
}
