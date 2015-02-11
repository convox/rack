package models

import (
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudwatch"
)

type Metrics struct {
	Cpu    float64
	Memory float64
	Disk   float64
}

func AppMetrics(app string) (*Metrics, error) {
	dimensions := []cloudwatch.Dimension{
		cloudwatch.Dimension{Name: aws.String("App"), Value: aws.String(app)},
		cloudwatch.Dimension{Name: aws.String("Process"), Value: aws.String("<all>")},
		cloudwatch.Dimension{Name: aws.String("InstanceId"), Value: aws.String("<all>")},
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
	dimensions := []cloudwatch.Dimension{
		cloudwatch.Dimension{Name: aws.String("App"), Value: aws.String(app)},
		cloudwatch.Dimension{Name: aws.String("Process"), Value: aws.String(process)},
		cloudwatch.Dimension{Name: aws.String("InstanceId"), Value: aws.String("<all>")},
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

func InstanceMetrics(app, process, instance string) (*Metrics, error) {
	dimensions := []cloudwatch.Dimension{
		cloudwatch.Dimension{Name: aws.String("App"), Value: aws.String(app)},
		cloudwatch.Dimension{Name: aws.String("Process"), Value: aws.String(process)},
		cloudwatch.Dimension{Name: aws.String("InstanceId"), Value: aws.String(instance)},
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

func getMetric(metric string, dimensions []cloudwatch.Dimension, span, precision int) ([]cloudwatch.Datapoint, error) {
	req := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: dimensions,
		EndTime:    time.Now(),
		MetricName: aws.String(metric),
		Namespace:  aws.String("Convox"),
		Period:     aws.Integer(precision * 60),
		StartTime:  time.Now().Add(time.Duration(-1*span) * time.Minute),
		Statistics: []string{"Average"},
	}

	res, err := Cloudwatch.GetMetricStatistics(req)

	if err != nil {
		return nil, err
	}

	return res.Datapoints, nil
}

func getLastAverage(data []cloudwatch.Datapoint) float64 {
	if len(data) < 1 {
		return 0
	} else {
		return *data[0].Average
	}
}
