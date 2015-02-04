package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudwatch"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/guillermo/go.procmeminfo"
)

type Metrics struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string
	AwsToken  string

	Tick time.Duration

	App      string
	Process  string
	Instance string
}

func (mm *Metrics) Monitor() {
	creds := aws.Creds(mm.AwsAccess, mm.AwsSecret, mm.AwsToken)
	cw := cloudwatch.New(creds, mm.AwsRegion, nil)

	for _ = range time.Tick(mm.Tick) {
		data := &cloudwatch.PutMetricDataInput{
			MetricData: []cloudwatch.MetricDatum{
				// mm.metricCPU(),
				mm.metricMemory(),
				mm.metricDisk(),
			},
			Namespace: aws.String("Convox"),
		}

		fmt.Printf("data %+v\n", data)
		continue

		fmt.Println("uploading cloudwatch metrics")
		err := cw.PutMetricData(data)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
	}
}

func (mm *Metrics) metricDimensions() []cloudwatch.Dimension {
	return []cloudwatch.Dimension{
		cloudwatch.Dimension{
			Name:  aws.String("App"),
			Value: aws.String(mm.App),
		},
		cloudwatch.Dimension{
			Name:  aws.String("Process"),
			Value: aws.String(mm.Process),
		},
		cloudwatch.Dimension{
			Name:  aws.String("InstanceId"),
			Value: aws.String(mm.Instance),
		},
	}
}

// func (mm *Metrics) metricCPU() cloudwatch.MetricDatum {
//   return cloudwatch.MetricDatum{
//     Dimensions: mm.metricDimensions(),
//     MetricName: aws.String("CpuUtilization"),
//     Timestamp:  time.Now(),
//     Unit:       aws.String("Percent"),
//     Value:      aws.Double(pct),
//   }
// }

func (mm *Metrics) metricMemory() cloudwatch.MetricDatum {
	meminfo := &procmeminfo.MemInfo{}
	meminfo.Update()

	pct := float64(meminfo.Used()) / float64(meminfo.Total()) * 100

	if pct == 0 {
		pct = 0.01
	}

	return cloudwatch.MetricDatum{
		Dimensions: mm.metricDimensions(),
		MetricName: aws.String("MemoryUtilization"),
		Timestamp:  time.Now(),
		Unit:       aws.String("Percent"),
		Value:      aws.Double(pct),
	}
}

func (mm *Metrics) metricDisk() cloudwatch.MetricDatum {
	usage, _ := exec.Command("bash", "-c", "df / | tail -n 1 | awk '{print $5}'").CombinedOutput()
	pct, _ := strconv.Atoi(string(usage[0 : len(usage)-2]))

	return cloudwatch.MetricDatum{
		Dimensions: mm.metricDimensions(),
		MetricName: aws.String("DiskUtilization"),
		Timestamp:  time.Now(),
		Unit:       aws.String("Percent"),
		Value:      aws.Double(float64(pct)),
	}
}
