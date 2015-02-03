package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/convox/agent/monitor"
)

type StringSlice []string

func (ss *StringSlice) String() string {
	return ""
}

func (ss *StringSlice) Set(v string) error {
	*ss = append(*ss, v)
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-log <file>] [-log <file>] [-cloudwatch <group>] [-kinesis <stream>] <autoscalegroup>\n", os.Args[0])
		flag.PrintDefaults()
	}

	logs := StringSlice{}
	flag.Var(&logs, "log", "log file")

	cwgroup := flag.String("cwgroup", "", "cloudwatch log group")
	cwstream := flag.String("cwstream", "", "cloudwatch log stream")
	kinesis := flag.String("kinesis", "", "kinesis stream")

	region := flag.String("region", "us-east-1", "aws region")
	access := flag.String("access", os.Getenv("AWS_ACCESS"), "aws access id")
	secret := flag.String("secret", os.Getenv("AWS_SECRET"), "aws secret key")

	flag.Parse()

	if len(flag.Args()) < 2 {
		flag.Usage()
		os.Exit(0)
	}

	asg := flag.Args()[0]
	instance := flag.Args()[1]

	mm := &monitor.Memory{
		AwsRegion:      *region,
		AwsAccess:      *access,
		AwsSecret:      *secret,
		Tick:           60 * time.Second,
		AutoScaleGroup: asg,
		InstanceId:     instance,
	}
	go mm.Monitor()

	lm := &monitor.Logs{
		AwsRegion:        *region,
		AwsAccess:        *access,
		AwsSecret:        *secret,
		Tick:             2 * time.Second,
		Logs:             logs,
		CloudwatchGroup:  *cwgroup,
		CloudwatchStream: *cwstream,
		Kinesis:          *kinesis,
	}
	go lm.Monitor()

	for {
		time.Sleep(10 * time.Minute)
	}
}
