package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	docker "github.com/convox/agent/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Monitor struct {
	client     *docker.Client
	envs       map[string]map[string]string
	instanceId string
	image      string
	lock       sync.Mutex
	lines      map[string][][]byte
}

func NewMonitor() *Monitor {
	client, err := docker.NewClient(os.Getenv("DOCKER_HOST"))

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("monitor new region=%s kinesis=%s log_group=%s\n", os.Getenv("AWS_REGION"), os.Getenv("KINESIS"), os.Getenv("LOG_GROUP"))

	return &Monitor{
		client:     client,
		envs:       make(map[string]map[string]string),
		lines:      make(map[string][][]byte),
		instanceId: GetInstanceId(),
		image:      "convox/agent", // also set during handleRunning
	}
}

func (m *Monitor) logEvent(id, message string) {
	env := m.envs[id]
	stream := env["KINESIS"]

	if stream != "" {
		m.addLine(stream, []byte(fmt.Sprintf("%s %s %s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, m.image, message)))
	}
}
