package monitor

import (
	"fmt"
	"os"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/cloudwatch"
	"github.com/guillermo/go.procmeminfo"
)

type Memory struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string

	Tick time.Duration

	AutoScaleGroup string
	InstanceId     string
}

func (mm *Memory) Monitor() {
	creds := aws.Creds(mm.AwsAccess, mm.AwsSecret, "")
	cw := cloudwatch.New(creds, mm.AwsRegion, nil)

	meminfo := &procmeminfo.MemInfo{}

	for _ = range time.Tick(mm.Tick) {
		meminfo.Update()
		pct := float64(meminfo.Used()) / float64(meminfo.Total()) * 100

		if pct == 0 {
			continue
		}

		data := &cloudwatch.PutMetricDataInput{
			MetricData: []cloudwatch.MetricDatum{
				cloudwatch.MetricDatum{
					Dimensions: []cloudwatch.Dimension{
						cloudwatch.Dimension{
							Name:  aws.String("AutoScalingGroupName"),
							Value: aws.String(mm.AutoScaleGroup),
						},
						cloudwatch.Dimension{
							Name:  aws.String("InstanceId"),
							Value: aws.String(mm.InstanceId),
						},
					},
					MetricName: aws.String("MemoryUtilization"),
					Timestamp:  time.Now(),
					Unit:       aws.String("Percent"),
					Value:      aws.Double(pct),
				},
			},
			Namespace: aws.String("Convox"),
		}

		fmt.Println("uploading cloudwatch metrics")
		err := cw.PutMetricData(data)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
	}
}
