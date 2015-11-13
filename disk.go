package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
)

// Monitor Disk Metrics for Instance
//
// Inspired by the techniques and Perl scripts in the CloudWatch Developer Guide
// http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/mon-scripts.html
//
// $./mon-put-instance-data.pl --swap-util --swap-used --disk-path / --disk-space-util --disk-space-used --disk-space-avail  --verify --verbose
// SwapUtilization: 0 (Percent)
// SwapUsed: 0 (Megabytes)
// DiskSpaceUtilization [/]: 23.3918103617163 (Percent)
// DiskSpaceUsed [/]: 6.87773513793945 (Gigabytes)
// DiskSpaceAvailable [/]: 22.2089805603027 (Gigabytes)
// No credential methods are specified. Trying default IAM role.
// Using IAM role <convox-IamRole-2B1GK98KX6BX>
// Endpoint: https://monitoring.us-west-2.amazonaws.com
//
// Payload: {"MetricData":[{"Timestamp":1447269869,"Dimensions":[{"Value":"i-287d9cf2","Name":"InstanceId"}],"Value":0,"Unit":"Percent","MetricName":"SwapUtilization"},{"Timestamp":1447269869,"Dimensions":[{"Value":"i-287d9cf2","Name":"InstanceId"}],"Value":0,"Unit":"Megabytes","MetricName":"SwapUsed"},{"Timestamp":1447269869,"Dimensions":[{"Value":"/dev/xvda1","Name":"Filesystem"},{"Value":"i-287d9cf2","Name":"InstanceId"},{"Value":"/","Name":"MountPath"}],"Value":23.3918103617163,"Unit":"Percent","MetricName":"DiskSpaceUtilization"},{"Timestamp":1447269869,"Dimensions":[{"Value":"/dev/xvda1","Name":"Filesystem"},{"Value":"i-287d9cf2","Name":"InstanceId"},{"Value":"/","Name":"MountPath"}],"Value":6.87773513793945,"Unit":"Gigabytes","MetricName":"DiskSpaceUsed"},{"Timestamp":1447269869,"Dimensions":[{"Value":"/dev/xvda1","Name":"Filesystem"},{"Value":"i-287d9cf2","Name":"InstanceId"},{"Value":"/","Name":"MountPath"}],"Value":22.2089805603027,"Unit":"Gigabytes","MetricName":"DiskSpaceAvailable"}],"Namespace":"System/Linux","__type":"com.amazonaws.cloudwatch.v2010_08_01#PutMetricDataInput"}
//
// Currently this only accurrately reports root disk usage on the Amazon ECS AMI, not Docker Machine and boot2docker
func MonitorDisk() {
	instance := GetInstanceId()

	fmt.Printf("disk monitor instance=%s\n", instance)

	stream := os.Getenv("KINESIS")

	// If no Kinesis stream to report to, no reason to calculate metrics
	if stream == "" {
		log.Printf("error: no rack KINESIS stream name is set\n")
		return
	}

	// On the ECS AMI /cgroup is on the root partition (/dev/xvda1)
	// However on boot2docker /cgroup is is a tmpfs
	// There is almost certainly a better way to introspect the root partition on all environments
	path := "/cgroup"

	for _ = range time.Tick(5 * time.Minute) {
		// https://github.com/StalkR/goircbot/blob/master/lib/disk/space_unix.go
		s := syscall.Statfs_t{}
		err := syscall.Statfs(path, &s)

		if err != nil {
			log.Printf("error: %s\n", err)
			continue
		}

		total := int(s.Bsize) * int(s.Blocks)
		free := int(s.Bsize) * int(s.Bfree)

		var avail, used, util float64
		avail = (float64)(free) / 1024 / 1024 / 1024
		used = (float64)(total-free) / 1024 / 1024 / 1024
		util = used / (used + avail) * 100

		log := fmt.Sprintf("disk monitor instance=%s utilization=%.2f%% used=%.4fG available=%.4fG\n", instance, util, used, avail)

		fmt.Print(log)
		err = PutRecord(stream, fmt.Sprintf("agent: %s", log))

		if err != nil {
			fmt.Printf("error: %s\n", err)
		}

		// If disk is over 80.0 full, delete docker containers and images
		// in attempt to reclaim space
		if util > 80.0 {
			RemoveDockerArtifacts()
		}
	}
}

// grep dmesg for file system error strings
// if grep exits 0 it was a match so we mark the instance unhealthy
// if grep exits 1 there was no match so we carry on
func MonitorDmesg() {
	instance := GetInstanceId()

	fmt.Printf("dmesg monitor instance=%s\n", instance)

	for _ = range time.Tick(5 * time.Minute) {
		cmd := exec.Command("sh", "-c", `dmesg | grep "Remounting filesystem read-only"`)
		out, err := cmd.CombinedOutput()

		// grep returned 0
		if err == nil {
			log := fmt.Sprintf("dmesg monitor instance=%s unhealthy=true msg=%q\n", instance, out)
			fmt.Print(log)

			stream := os.Getenv("KINESIS")

			if stream != "" {
				err = PutRecord(stream, fmt.Sprintf("agent: %s", log))

				if err != nil {
					fmt.Printf("error: %s\n", err)
					continue
				}
			}

			AutoScaling := autoscaling.New(&aws.Config{})

			_, err := AutoScaling.SetInstanceHealth(&autoscaling.SetInstanceHealthInput{
				HealthStatus:             aws.String("Unhealthy"),
				InstanceID:               aws.String(instance),
				ShouldRespectGracePeriod: aws.Boolean(true),
			})

			if err != nil {
				fmt.Printf("%+v\n", err)
			}
		}
	}
}

// Get an instance identifier
// On EC2 use the meta-data API to get an instance id
// Fall back to system hostname if unavailable
func GetInstanceId() string {
	hostname, err := os.Hostname()

	if err != nil {
		fmt.Printf("error: %s\n", err)
		hostname = "unknown-host"
	}

	resp, err := http.Get("http://169.254.169.254/latest/meta-data/instance-id")

	if err != nil {
		fmt.Printf("error: %s\n", err)
		return hostname
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("error: %s\n", err)
		return hostname
	}

	return string(body)
}

func PutRecord(stream, s string) error {
	Kinesis := kinesis.New(&aws.Config{})

	record := &kinesis.PutRecordInput{
		Data:         []byte(s),
		StreamName:   aws.String(stream),
		PartitionKey: aws.String(string(time.Now().UnixNano())),
	}

	_, err := Kinesis.PutRecord(record)

	if err != nil {
		return err
	}

	fmt.Printf("disk monitor upload to=kinesis stream=%q lines=1\n", stream)

	return nil
}

// Force remove docker containers, volumes and images
// This is a quick and dirty way to remove everything but running containers their images
// This will blow away build or run cache but hopefully preserve
// disk space.
func RemoveDockerArtifacts() {
	instance := GetInstanceId()

	prefix := fmt.Sprintf("remove_docker monitor instance=%s", instance)

	run(prefix, `docker rm -v $(docker ps -a -q)`)
	run(prefix, `docker rmi -f $(docker images -a -q)`)
}

// Blindly run a shell command and log its output and error
func run(log_prefix, cmd string) {
	fmt.Printf("%s cmd=%q\n", log_prefix, cmd)

	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		fmt.Printf("%s out=%q\n", log_prefix, l)
	}

	if err != nil {
		fmt.Printf("%s error=%q\n", log_prefix, err)
	}
}
