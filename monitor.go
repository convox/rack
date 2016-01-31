package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger"
	docker "github.com/convox/agent/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Monitor struct {
	client *docker.Client

	envs map[string]map[string]string

	agentId    string
	agentImage string

	instanceId string

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

		agentImage: "convox/agent", // also set during handleRunning

		instanceId: GetInstanceId(),

		lines:   make(map[string][][]byte),
		loggers: make(map[string]logger.Logger),
	}
}

// Get an instance identifier
// On EC2 use the meta-data API to get an instance id
// Fall back to system hostname written in 'i-12345678' style if unavailable
func GetInstanceId() string {
	hostname, err := os.Hostname()

	if err != nil {
		fmt.Printf("error: %s\n", err)
		hostname = "hosterr"
	}

	if len(hostname) > 8 {
		hostname = hostname[0:8]
	}

	hostname = fmt.Sprintf("i-%s", hostname)

	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	resp, err := client.Get("http://169.254.169.254/latest/meta-data/instance-id")

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

func (m *Monitor) logAppEvent(id, message string) {
	msg := []byte(fmt.Sprintf("%s %s %s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, m.agentImage, message))

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

func (m *Monitor) logSystemEvent(message string) {
	m.logAppEvent(m.agentId, message)
}
