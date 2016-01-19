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
	"github.com/convox/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger"
	"github.com/convox/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger/awslogs"
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

	fmt.Printf("monitor new region=%s kinesis=%s\n", os.Getenv("AWS_REGION"), os.Getenv("KINESIS"))

	return &Monitor{
		client:     client,
		envs:       make(map[string]map[string]string),
		lines:      make(map[string][][]byte),
		instanceId: GetInstanceId(),
		image:      "convox/agent", // also set during handleRunning
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
			m.image = img
			fmt.Printf("monitor event id=%s status=skipped\n", shortId)
			continue
		}

		fmt.Printf("monitor event id=%s status=started\n", shortId)
		m.handleStart(container.ID)
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
	m.logEvent(id, fmt.Sprintf("Starting process %s", id[0:12]))
}

func (m *Monitor) handleDie(id string) {
	// While we could remove a container and volumes on this event
	// It seems like explicitly doing a `docker run --rm` is the best way
	// to state this intent.
	m.logEvent(id, fmt.Sprintf("Dead process %s", id[0:12]))
}

func (m *Monitor) handleKill(id string) {
	m.logEvent(id, fmt.Sprintf("Stopped process %s via SIGKILL", id[0:12]))
}

func (m *Monitor) handleStart(id string) {
	container, env, err := m.inspectContainer(id)

	if err != nil {
		log.Printf("error: %s\n", err)
		return
	}

	m.envs[id] = env

	m.updateCgroups(id, env)

	go m.subscribeLogs(container, env)
}

func (m *Monitor) handleStop(id string) {
	m.logEvent(id, fmt.Sprintf("Stopped process %s via SIGTERM", id[0:12]))
}

func (m *Monitor) inspectContainer(id string) (*docker.Container, map[string]string, error) {
	env := map[string]string{}

	container, err := m.client.InspectContainer(id)

	if err != nil {
		log.Printf("error: %s\n", err)
		return container, env, err
	}

	for _, e := range container.Config.Env {
		parts := strings.SplitN(e, "=", 2)

		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return container, env, nil
}

func (m *Monitor) logEvent(id, message string) {
	env := m.envs[id]
	stream := env["KINESIS"]

	if stream != "" {
		m.addLine(stream, []byte(fmt.Sprintf("%s %s %s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, m.image, message)))
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

func (m *Monitor) subscribeLogs(container *docker.Container, env map[string]string) {
	id := container.ID

	kinesis := env["KINESIS"]
	logGroup := env["LOG_GROUP"]
	process := env["PROCESS"]
	release := env["RELEASE"]

	logResource := kinesis
	if logResource == "" {
		logResource = logGroup
	}

	if logResource == "" {
		fmt.Printf("agent _fn=subscribeLogs at=skip id=%s kinesis=%s logGroup=%s process=%s instanceId=%s\n", id, kinesis, logGroup, process, m.instanceId)
		return
	}

	fmt.Printf("agent _fn=subscribeLogs at=start id=%s kinesis=%s process=%s logGroup=%s instanceId=%s\n", id, kinesis, logGroup, process, m.instanceId)

	// extract app name from kinesis or logGroup
	// myapp-staging-Kinesis-L6MUKT1VH451 -> myapp-staging
	app := ""

	parts := strings.Split(logResource, "-")
	if len(parts) > 2 {
		app = strings.Join(parts[0:len(parts)-2], "-") // drop -Kinesis-YXXX
	}

	// create a an awslogger and associated CloudWatch Logs LogGroup
	// if this doesn't error, write to the logger in the scanner loop
	awslogger, aerr := m.StartAWSLogger(container, logGroup)

	if aerr != nil {
		fmt.Printf("ERROR: %+v\n", aerr)
	}

	r, w := io.Pipe()

	go func(prefix string, r io.ReadCloser) {
		defer r.Close()

		fmt.Printf("agent _fn=subscribeLogs.Scan at=start prefix=%s\n", prefix)

		scanner := bufio.NewScanner(r)

		for scanner.Scan() {
			text := scanner.Text()

			line := []byte(fmt.Sprintf("%s %s %s/%s:%s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, app, process, release, text))

			if kinesis != "" {
				m.addLine(kinesis, line)
			}

			if aerr == nil {
				awslogger.Log(&logger.Message{
					ContainerID: container.ID,
					Line:        line,
					Timestamp:   time.Now(),
				})
			}
		}

		if scanner.Err() != nil {
			fmt.Printf("agent _fn=subscribeLogs.Scan dim#process=agent dim#instanceId=%s count#scanner.error=1 msg=%q\n", m.instanceId, scanner.Err().Error())
		}

		fmt.Printf("agent _fn=subscribeLogs.Scan at=return prefix=%s\n", prefix)
	}(process, r)

	// tail docker logs and write to pipe
	since := time.Unix(0, 0).Unix()

	for {
		fmt.Printf("agent _fn=subscribeLogs id=%s kinesis=%s logGroup=%s process=%s since=%d dim#process=agent dim#instanceId=%s count#docker.Logs.start=1\n", id, kinesis, logGroup, process, since, m.instanceId)

		err := m.client.Logs(docker.LogsOptions{
			Since:        since,
			Container:    id,
			Follow:       true,
			Stdout:       true,
			Stderr:       true,
			Tail:         "all",
			RawTerminal:  false,
			OutputStream: w,
			ErrorStream:  w,
		})

		since = time.Now().Unix() // update cursor to now in anticipation of retry

		if err != nil {
			fmt.Printf("agent _fn=subscribeLogs id=%s kinesis=%s logGroup=%s process=%s dim#process=agent dim#instanceId=%s count#docker.Logs.error=1 msg=%q\n", id, kinesis, logGroup, process, m.instanceId, err.Error())
		}

		container, err := m.client.InspectContainer(id)

		if err != nil {
			fmt.Printf("agent _fn=subscribeLogs id=%s kinesis=%s logGroup=%s process=%s dim#process=agent dim#instanceId=%s count#docker.InspectContainer.error=1 msg=%q\n", id, kinesis, logGroup, process, m.instanceId, err.Error())
			break
		}

		if container.State.Running == false {
			break
		}
	}

	w.Close()

	fmt.Printf("agent _fn=subscribeLogs at=return id=%s kinesis=%s logGroup=%s process=%s instanceId=%s\n", id, kinesis, logGroup, process, m.instanceId)
}

func (m *Monitor) StartAWSLogger(container *docker.Container, logGroup string) (logger.Logger, error) {
	ctx := logger.Context{
		Config: map[string]string{
			"awslogs-group": logGroup,
		},
		ContainerID:         container.ID,
		ContainerName:       container.Name,
		ContainerEntrypoint: container.Path,
		ContainerArgs:       container.Args,
		ContainerImageID:    container.Image,
		ContainerImageName:  container.Config.Image,
		ContainerCreated:    container.Created,
		ContainerEnv:        container.Config.Env,
		ContainerLabels:     container.Config.Labels,
	}

	return awslogs.New(ctx)
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
				fmt.Printf("agent _fn=streamLogs stream=%s dim#process=agent dim#instanceId=%s count#Kinesis.PutRecords.error=1 msg=%q\n", stream, m.instanceId, err.Error())
			}

			errorCount := 0
			errorMsg := ""

			for _, r := range res.Records {
				if r.ErrorCode != nil {
					errorCount += 1
					errorMsg = fmt.Sprintf("%s - %s", *r.ErrorCode, *r.ErrorMessage)
				}
			}

			fmt.Printf("agent _fn=streamLogs stream=%s dim#process=agent dim#instanceId=%s count#Kinesis.PutRecords.records=%d count#Kinesis.PutRecords.records.errors=%d msg=%q\n", stream, m.instanceId, len(res.Records), errorCount, errorMsg)
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
