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
	m.logSystemEvent("dmesg monitor at=start", "")

	for _ = range time.Tick(MONITOR_INTERVAL) {
		cmd := exec.Command("sh", "-c", `dmesg | grep "Remounting filesystem read-only"`)
		out, err := cmd.CombinedOutput()

		// grep returned 0
		if err == nil {
			m.logSystemEvent("dmesg monitor at=error", fmt.Sprintf("dim#system=dmesg count#AutoScaling.SetInstanceHealth=1 out=%q", out))

			AutoScaling := autoscaling.New(&aws.Config{})

			_, err := AutoScaling.SetInstanceHealth(&autoscaling.SetInstanceHealthInput{
				HealthStatus:             aws.String("Unhealthy"),
				InstanceId:               aws.String(m.instanceId),
				ShouldRespectGracePeriod: aws.Bool(true),
			})

			if err != nil {
				m.logSystemEvent("dmesg monitor at=error", fmt.Sprintf("dim#system=dmesg count#AutoScaling.SetInstanceHealth.error=1 err=%q", err))
			}
		} else {
			m.logSystemEvent("dmesg monitor at=ok", "")
		}
	}
}
