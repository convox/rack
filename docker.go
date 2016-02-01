package main

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
)

// interact with dockerd for docker errors
// if `docker ps` exits non-zero we mark the instance unhealthy
func (m *Monitor) Docker() {
	m.logSystemEvent("docker monitor at=start", "")

	for _ = range time.Tick(MONITOR_INTERVAL) {
		cmd := exec.Command("docker", "ps")

		if err := cmd.Start(); err != nil {
			m.logSystemEvent("docker monitor at=error", fmt.Sprintf("dim#system=docker count#Command.Start.error=1 err=%q", err))
			continue
		}

		timer := time.AfterFunc(10*time.Second, func() {
			cmd.Process.Kill()
		})

		err := cmd.Wait()
		timer.Stop()

		// docker ps returned non-zero
		if err != nil {
			m.logSystemEvent("docker monitor at=error", fmt.Sprintf("dim#system=docker count#AutoScaling.SetInstanceHealth=1 err=%q", err))

			AutoScaling := autoscaling.New(&aws.Config{})

			_, err := AutoScaling.SetInstanceHealth(&autoscaling.SetInstanceHealthInput{
				HealthStatus:             aws.String("Unhealthy"),
				InstanceId:               aws.String(m.instanceId),
				ShouldRespectGracePeriod: aws.Bool(true),
			})

			if err != nil {
				m.logSystemEvent("docker monitor at=error", fmt.Sprintf("dim#system=docker count#AutoScaling.SetInstanceHealth.error=1 err=%q", err))
			}
		} else {
			m.logSystemEvent("docker monitor at=ok", "")
		}
	}
}
