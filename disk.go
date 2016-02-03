package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/docker/go-units"
)

// Monitor Disk Metrics for Instance
// Currently this only accurrately reports disk usage on the Amazon ECS AMI and the devicemapper driver
// not Docker Machine, boot2docker and aufs driver
func (m *Monitor) Disk() {
	m.logSystemMetric("disk at=start", "", true)

	counter := 0

	for _ = range time.Tick(MONITOR_INTERVAL) {
		info, err := m.client.Info()

		if err != nil {
			m.logSystemMetric("disk at=error", fmt.Sprintf("err=%q", err), true)
		}

		status := [][]string{}

		err = info.GetJSON("DriverStatus", &status)

		if err != nil {
			m.logSystemMetric("disk at=error", fmt.Sprintf("err=%q", err), true)
			continue
		}

		var avail, total, used int64

		for _, v := range status {
			if v[0] == "Data Space Available" {
				avail, err = units.FromHumanSize(v[1])

				if err != nil {
					m.logSystemMetric("disk at=error", fmt.Sprintf("err=%q", err), true)
					continue
				}
			}

			if v[0] == "Data Space Total" {
				total, err = units.FromHumanSize(v[1])

				if err != nil {
					m.logSystemMetric("disk at=error", fmt.Sprintf("err=%q", err), true)
					continue
				}
			}

			if v[0] == "Data Space Used" {
				used, err = units.FromHumanSize(v[1])

				if err != nil {
					m.logSystemMetric("disk at=error", fmt.Sprintf("err=%q", err), true)
					continue
				}
			}
		}

		if total == 0 {
			m.logSystemMetric("disk at=skip", fmt.Sprintf("driver=%s", m.dockerDriver), true)
			continue
		}

		var a, t, u, util float64
		a = float64(avail) / 1000 / 1000 / 1000
		t = float64(total) / 1000 / 1000 / 1000
		u = float64(used) / 1000 / 1000 / 1000
		util = float64(used) / float64(total) * 100

		m.logSystemMetric("disk", fmt.Sprintf("dim#instanceId=%s sample#disk.available=%.4fgB sample#disk.total=%.4fgB sample#disk.used=%.4fgB sample#disk.utilization=%.2f%%", m.instanceId, a, t, u, util), true)

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
	m.logSystemMetric("disk", "count#docker.rmi=1", true)

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
