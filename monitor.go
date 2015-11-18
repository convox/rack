package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
	docker "github.com/convox/agent/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Monitor struct {
	client     *docker.Client
	envs       map[string]map[string]string
	instanceId string
	lock       sync.Mutex
	lines      map[string][][]byte
}

func NewMonitor() *Monitor {
	client, err := docker.NewClient(os.Getenv("DOCKER_HOST"))

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("monitor new region=%s kinesis=%s\n", os.Getenv("AWS_REGION"), os.Getenv("KINESIS"))

	return &Monitor{
		client:     client,
		envs:       make(map[string]map[string]string),
		lines:      make(map[string][][]byte),
		instanceId: GetInstanceId(),
	}
}

func (m *Monitor) Listen() {
	m.handleRunning()
	m.handleExited()

	ch := make(chan *docker.APIEvents)

	go m.handleEvents(ch)
	go m.streamLogs()

	m.client.AddEventListener(ch)

	for {
		time.Sleep(60 * time.Second)
	}
}

// List already running containers and subscribe and stream logs
func (m *Monitor) handleRunning() {
	containers, err := m.client.ListContainers(docker.ListContainersOptions{})

	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		shortId := container.ID[0:12]

		// Don't subscribe and stream logs from the agent container itself
		img := container.Image

		if strings.HasPrefix(img, "convox/agent") || strings.HasPrefix(img, "agent/agent") {
			fmt.Printf("monitor event id=%s status=skipped\n", shortId)
			continue
		}

		fmt.Printf("monitor event id=%s status=created\n", shortId)
		m.handleCreate(container.ID)
	}
}

// List already exiteded containers and remove
func (m *Monitor) handleExited() {
	containers, err := m.client.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"status": []string{"exited"},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		shortId := container.ID[0:12]

		fmt.Printf("monitor event id=%s status=died\n", shortId)
		m.handleDie(container.ID)
	}
}

func (m *Monitor) handleEvents(ch chan *docker.APIEvents) {
	for event := range ch {

		shortId := event.ID

		if len(shortId) > 12 {
			shortId = shortId[0:12]
		}

		fmt.Printf("monitor event id=%s status=%s time=%d\n", shortId, event.Status, event.Time)

		switch event.Status {
		case "create":
			m.handleCreate(event.ID)
		case "die":
			m.handleDie(event.ID)
		case "kill":
			m.handleKill(event.ID)
		case "start":
			m.handleStart(event.ID)
		case "stop":
			m.handleStop(event.ID)
		}
	}
}

func (m *Monitor) handleCreate(id string) {
	env, err := m.inspectContainerEnv(id)

	if err != nil {
		log.Printf("error: %s\n", err)
		return
	}

	m.envs[id] = env

	go m.subscribeLogs(id, env["KINESIS"], env["PROCESS"], env["RELEASE"])
}

func (m *Monitor) handleDie(id string) {
	// While we could remove a container and volumes on this event
	// It seems like explicitly doing a `docker run --rm` is the best way
	// to state this intent.
}

func (m *Monitor) handleKill(id string) {
	m.logEvent(id, "Stopping container with SIGKILL")
}

func (m *Monitor) handleStart(id string) {
	m.updateCgroups(id, m.envs[id])
}

func (m *Monitor) handleStop(id string) {
	m.logEvent(id, "Stopping container with SIGTERM")
}

func (m *Monitor) inspectContainerEnv(id string) (map[string]string, error) {
	env := map[string]string{}

	container, err := m.client.InspectContainer(id)

	if err != nil {
		log.Printf("error: %s\n", err)
		return env, err
	}

	for _, e := range container.Config.Env {
		parts := strings.SplitN(e, "=", 2)

		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return env, nil
}

func (m *Monitor) logEvent(id, message string) {
	env := m.envs[id]
	stream := env["KINESIS"]

	if stream != "" {
		m.addLine(stream, []byte(fmt.Sprintf("%s [%s/%s/%s]: %s", "convox/agent", m.instanceId, id[0:12], env["RELEASE"], message)))
	}
}

// Modify the container cgroup to enable swap if SWAP=1 is set
// Currently this only works on the Amazon ECS AMI, not Docker Machine and boot2docker
// until a better strategy for knowing where the cgroup mount is implemented
func (m *Monitor) updateCgroups(id string, env map[string]string) {
	if env["SWAP"] == "1" {
		shortId := id[0:12]

		bytes := "18446744073709551615"

		fmt.Printf("monitor cgroups id=%s cgroup=memory.memsw.limit_in_bytes value=%s\n", shortId, bytes)
		err := ioutil.WriteFile(fmt.Sprintf("/cgroup/memory/docker/%s/memory.memsw.limit_in_bytes", id), []byte(bytes), 0644)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}

		fmt.Printf("monitor cgroups id=%s cgroup=memory.soft_limit_in_bytes value=%s\n", shortId, bytes)
		err = ioutil.WriteFile(fmt.Sprintf("/cgroup/memory/docker/%s/memory.soft_limit_in_bytes", id), []byte(bytes), 0644)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}

		fmt.Printf("monitor cgroups id=%s cgroup=memory.limit_in_bytes value=%s\n", shortId, bytes)
		err = ioutil.WriteFile(fmt.Sprintf("/cgroup/memory/docker/%s/memory.limit_in_bytes", id), []byte(bytes), 0644)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
	}
}

func (m *Monitor) subscribeLogs(id, stream, process, release string) {
	if stream == "" {
		return
	}

	time.Sleep(500 * time.Millisecond)

	r, w := io.Pipe()

	go func(prefix string, r io.ReadCloser) {
		defer r.Close()

		scanner := bufio.NewScanner(r)

		for scanner.Scan() {
			m.addLine(stream, []byte(fmt.Sprintf("%s [%s/%s/%s]: %s", process, m.instanceId, id[0:12], release, scanner.Text())))
		}

		if scanner.Err() != nil {
			log.Printf("error: %s\n", scanner.Err())
		}
	}(process, r)

	err := m.client.Logs(docker.LogsOptions{
		Container:    id,
		Follow:       true,
		Stdout:       true,
		Stderr:       true,
		Tail:         "all",
		RawTerminal:  false,
		OutputStream: w,
		ErrorStream:  w,
	})

	if err != nil {
		log.Printf("error: %s\n", err)
	}

	w.Close()
}

func (m *Monitor) streamLogs() {
	Kinesis := kinesis.New(&aws.Config{})

	for _ = range time.Tick(100 * time.Millisecond) {
		for _, stream := range m.streams() {
			l := m.getLines(stream)

			if l == nil {
				continue
			}

			records := &kinesis.PutRecordsInput{
				Records:    make([]*kinesis.PutRecordsRequestEntry, len(l)),
				StreamName: aws.String(stream),
			}

			for i, line := range l {
				records.Records[i] = &kinesis.PutRecordsRequestEntry{
					Data:         line,
					PartitionKey: aws.String(string(time.Now().UnixNano())),
				}
			}

			res, err := Kinesis.PutRecords(records)

			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
			}

			for _, r := range res.Records {
				if r.ErrorCode != nil {
					fmt.Printf("error: %s\n", *r.ErrorCode)
				}
			}

			fmt.Printf("monitor upload to=kinesis stream=%q lines=%d\n", stream, len(res.Records))
		}
	}
}

func (m *Monitor) addLine(stream string, data []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.lines[stream] = append(m.lines[stream], data)
}

func (m *Monitor) getLines(stream string) [][]byte {
	m.lock.Lock()
	defer m.lock.Unlock()

	nl := len(m.lines[stream])

	if nl == 0 {
		return nil
	}

	if nl > 500 {
		nl = 500
	}

	ret := make([][]byte, nl)
	copy(ret, m.lines[stream])
	m.lines[stream] = m.lines[stream][nl:]

	return ret
}

func (m *Monitor) streams() []string {
	m.lock.Lock()
	defer m.lock.Unlock()

	streams := make([]string, len(m.lines))
	i := 0

	for key, _ := range m.lines {
		streams[i] = key
		i += 1
	}

	return streams
}
