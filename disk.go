package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"syscall"
	"time"
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
func (m *Monitor) Disk() {
	fmt.Printf("disk monitor instance=%s\n", m.instanceId)

	// On the ECS AMI /cgroup is on the root partition (/dev/xvda1)
	// However on boot2docker /cgroup is is a tmpfs
	// There is almost certainly a better way to introspect the root partition on all environments
	path := "/cgroup"

	counter := 0

	for _ = range time.Tick(MONITOR_INTERVAL) {
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

		m.logSystemMetric("disk", fmt.Sprintf("sample#disk.utilization=%.2f%% sample#disk.used=%.4fgB sample#disk.available=%.4fgB", util, used, avail), true)

		// If disk is over 80.0 full, delete docker containers and images in attempt to reclaim space
		// Only do this every 12th tick (60 minutes)
		counter += 1
		if util > 80.0 && counter%12 == 0 {
			m.RemoveDockerArtifacts()
		}
	}
}

// Force remove docker containers, volumes and images
// This is a quick and dirty way to remove everything but running containers their images
// This will blow away build or run cache but hopefully preserve
// disk space.
func (m *Monitor) RemoveDockerArtifacts() {
	m.logSystemMetric("disk", "dim#system=Monitor.RemoveDockerArtifacts count#docker.rmi=1", true)

	m.run(`docker rm -v $(docker ps -a -q)`)
	m.run(`docker rmi -f $(docker images -a -q)`)
}

// Blindly run a shell command and log its output and error
func (m *Monitor) run(cmd string) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		m.logSystemMetric("disk run", fmt.Sprintf("%s cmd=%q out=%q", cmd, l), true)
	}

	if err != nil {
		m.logSystemMetric("disk run", fmt.Sprintf("error=%q", err), true)
	}
}
