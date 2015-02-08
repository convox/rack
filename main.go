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
		fmt.Fprintf(os.Stderr, "Usage: %s [-log <file>] [-log <file>] [-cloudwatch <group>] [-kinesis <stream>] <app> <process> <instance>\n", os.Args[0])
		flag.PrintDefaults()
	}

	logs := StringSlice{}
	flag.Var(&logs, "log", "log file")

	cwgroup := flag.String("cwgroup", "", "cloudwatch log group")
	cwstream := flag.String("cwstream", "", "cloudwatch log stream")
	kinesis := flag.String("kinesis", "", "kinesis stream")
	tick := flag.Int("tick", 30, "metric update interval")

	region := flag.String("region", "us-east-1", "aws region")
	access := flag.String("access", os.Getenv("AWS_ACCESS"), "aws access id")
	secret := flag.String("secret", os.Getenv("AWS_SECRET"), "aws secret key")
	token := flag.String("token", os.Getenv("AWS_TOKEN"), "aws token")

	flag.Parse()

	if len(flag.Args()) < 3 {
		flag.Usage()
		os.Exit(0)
	}

	app := flag.Args()[0]
	process := flag.Args()[1]
	instance := flag.Args()[2]

	mm := &monitor.Metrics{
		AwsRegion: *region,
		AwsAccess: *access,
		AwsSecret: *secret,
		AwsToken:  *token,
		Tick:      time.Duration(*tick) * time.Second,
		App:       app,
		Process:   process,
		Instance:  instance,
	}
	go mm.Monitor()

	lm := &monitor.Logs{
		AwsRegion:        *region,
		AwsAccess:        *access,
		AwsSecret:        *secret,
		AwsToken:         *token,
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
