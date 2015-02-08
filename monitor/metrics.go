package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/bitbucket.org/bertimus9/systemstat"
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
		data := &cloudwatch.PutMetricDataInput{Namespace: aws.String("Convox")}

		for _, d := range mm.metricCPU() {
			data.MetricData = append(data.MetricData, d)
		}

		for _, d := range mm.metricMemory() {
			data.MetricData = append(data.MetricData, d)
		}

		for _, d := range mm.metricDisk() {
			data.MetricData = append(data.MetricData, d)
		}

		fmt.Println("uploading cloudwatch metrics")
		err := cw.PutMetricData(data)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
	}
}

func (mm *Metrics) metricDimensions() [][]cloudwatch.Dimension {
	return [][]cloudwatch.Dimension{
		[]cloudwatch.Dimension{
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
		},
		[]cloudwatch.Dimension{
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
				Value: aws.String("<all>"),
			},
		},
		[]cloudwatch.Dimension{
			cloudwatch.Dimension{
				Name:  aws.String("App"),
				Value: aws.String(mm.App),
			},
			cloudwatch.Dimension{
				Name:  aws.String("Process"),
				Value: aws.String("<all>"),
			},
			cloudwatch.Dimension{
				Name:  aws.String("InstanceId"),
				Value: aws.String("<all>"),
			},
		},
	}
}

func (mm *Metrics) metricCPU() []cloudwatch.MetricDatum {
	s1 := systemstat.GetCPUSample()
	time.Sleep(2 * time.Second)
	s2 := systemstat.GetCPUSample()
	sample := systemstat.GetCPUAverage(s1, s2)
	pct := 100.0 - sample.IdlePct

	dimensions := mm.metricDimensions()
	data := make([]cloudwatch.MetricDatum, len(dimensions))

	for i, dim := range dimensions {
		data[i] = cloudwatch.MetricDatum{
			Dimensions: dim,
			MetricName: aws.String("CpuUtilization"),
			Timestamp:  time.Now(),
			Unit:       aws.String("Percent"),
			Value:      aws.Double(pct),
		}
	}

	return data
}

func (mm *Metrics) metricMemory() []cloudwatch.MetricDatum {
	meminfo := &procmeminfo.MemInfo{}
	meminfo.Update()

	pct := float64(meminfo.Used()) / float64(meminfo.Total()) * 100

	if pct == 0 {
		pct = 0.01
	}

	dimensions := mm.metricDimensions()
	data := make([]cloudwatch.MetricDatum, len(dimensions))

	for i, dim := range dimensions {
		data[i] = cloudwatch.MetricDatum{
			Dimensions: dim,
			MetricName: aws.String("MemoryUtilization"),
			Timestamp:  time.Now(),
			Unit:       aws.String("Percent"),
			Value:      aws.Double(pct),
		}
	}

	return data
}

func (mm *Metrics) metricDisk() []cloudwatch.MetricDatum {
	usage, _ := exec.Command("bash", "-c", "df / | tail -n 1 | awk '{print $5}'").CombinedOutput()
	pct, _ := strconv.Atoi(string(usage[0 : len(usage)-2]))

	dimensions := mm.metricDimensions()
	data := make([]cloudwatch.MetricDatum, len(dimensions))

	for i, dim := range dimensions {
		data[i] = cloudwatch.MetricDatum{
			Dimensions: dim,
			MetricName: aws.String("DiskUtilization"),
			Timestamp:  time.Now(),
			Unit:       aws.String("Percent"),
			Value:      aws.Double(float64(pct)),
		}
	}

	return data
}
