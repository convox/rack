package manifest

import (
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var Docker = func(args ...string) *exec.Cmd {
	return exec.Command("docker", args...)
}

func dockerHost() (host string) {
	host = "127.0.0.1"

	if h := os.Getenv("DOCKER_HOST"); h != "" {
		u, err := url.Parse(h)

		if err != nil {
			return
		}

		parts := strings.Split(u.Host, ":")
		host = parts[0]
	}

	return
}

func DockerHostExposedPorts() ([]int, error) {
	open := []int{}

	data, err := Docker("ps", "--format", "{{.ID}}").Output()

	if err != nil {
		return nil, err
	}

	for _, ps := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if ps == "" {
			continue
		}

		data, err := Docker("inspect", "--format", "{{json .NetworkSettings.Ports}}", ps).Output()

		if err != nil {
			return nil, err
		}

		var ports map[string][]struct {
			HostPort string
		}

		err = json.Unmarshal(data, &ports)

		if err != nil {
			return nil, err
		}

		for _, port := range ports {
			for _, m := range port {
				p, err := strconv.Atoi(m.HostPort)

				if err != nil {
					return nil, err
				}

				open = append(open, p)
			}
		}
	}

	return open, nil
}
