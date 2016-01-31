package main

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
)

// grep dmesg for file system error strings
// if grep exits 0 it was a match so we mark the instance unhealthy
// if grep exits 1 there was no match so we carry on
func (m *Monitor) Dmesg() {
	fmt.Printf("dmesg monitor instance=%s\n", m.instanceId)

	for _ = range time.Tick(MONITOR_INTERVAL) {
		cmd := exec.Command("sh", "-c", `dmesg | grep "Remounting filesystem read-only"`)
		out, err := cmd.CombinedOutput()

		// grep returned 0
		if err == nil {
			m.logSystemEvent("dmesg monitor", fmt.Sprintf("count#dmesg.unhealthy=1 msg=%q", out))

			AutoScaling := autoscaling.New(&aws.Config{})

			_, err := AutoScaling.SetInstanceHealth(&autoscaling.SetInstanceHealthInput{
				HealthStatus:             aws.String("Unhealthy"),
				InstanceId:               aws.String(m.instanceId),
				ShouldRespectGracePeriod: aws.Bool(true),
			})

			if err != nil {
				fmt.Printf("%+v\n", err)
			}
		}
	}
}
