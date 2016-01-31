package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger"
	docker "github.com/convox/agent/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Monitor struct {
	client *docker.Client

	envs map[string]map[string]string

	instanceId string
	image      string

	lock    sync.Mutex
	lines   map[string][][]byte
	loggers map[string]logger.Logger
}

func NewMonitor() *Monitor {
	client, err := docker.NewClient(os.Getenv("DOCKER_HOST"))

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("monitor new region=%s kinesis=%s log_group=%s\n", os.Getenv("AWS_REGION"), os.Getenv("KINESIS"), os.Getenv("LOG_GROUP"))

	return &Monitor{
		client: client,

		envs: make(map[string]map[string]string),

		instanceId: GetInstanceId(),
		image:      "convox/agent", // also set during handleRunning

		lines:   make(map[string][][]byte),
		loggers: make(map[string]logger.Logger),
	}
}

func (m *Monitor) logAppEvent(id, message string) {
	msg := []byte(fmt.Sprintf("%s %s %s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, m.image, message))

	if awslogger, ok := m.loggers[id]; ok {
		awslogger.Log(&logger.Message{
			ContainerID: id,
			Line:        msg,
			Timestamp:   time.Now(),
		})
	}

	if stream, ok := m.envs[id]["KINESIS"]; ok {
		m.addLine(stream, msg)
	}
}
