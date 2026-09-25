package aws

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const buildStuckTaskArn = "arn:aws:ecs:us-test-1:901416387788:task/cluster-test/50b8de99"

func stubProviderHandler(t *testing.T, h http.Handler) *Provider {
	t.Helper()
	t.Setenv("PROVIDER", "test")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-test-1")
	t.Setenv("AWS_SESSION_TOKEN", "")

	srv := httptest.NewServer(h)

	t.Cleanup(srv.Close)

	return &Provider{
		BuildCluster:   "cluster-test",
		Cluster:        "cluster-test",
		DynamoBuilds:   "convox-builds",
		Endpoint:       srv.URL,
		Rack:           "convox",
		Region:         "us-test-1",
		SettingsBucket: "convox-settings",
		SkipCache:      true,
	}
}

func stubProvider(t *testing.T, cycles ...awsutil.Cycle) (*Provider, *[]string) {
	t.Helper()

	inner := awsutil.NewHandler(cycles)
	bodies := []string{}

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %s", err)
			return
		}

		bodies = append(bodies, string(b))
		r.Body = io.NopCloser(bytes.NewReader(b))

		inner.ServeHTTP(w, r)
	}))

	return p, &bodies
}

func buildItem(status, reason, logs string) string {
	item := `"id": {"S": "B123"},
		"app": {"S": "httpd"},
		"created": {"S": "20160404.143416.178278576"},
		"status": {"S": "` + status + `"},
		"tags": {"B": "eyJ0YXNrIjoiYXJuOmF3czplY3M6dXMtdGVzdC0xOjkwMTQxNjM4Nzc4ODp0YXNrL2NsdXN0ZXItdGVzdC81MGI4ZGU5OSJ9"}`

	if reason != "" {
		item += `, "reason": {"S": "` + reason + `"}`
	}

	if logs != "" {
		item += `, "logs": {"S": "` + logs + `"}`
	}

	return `{"Item":{` + item + `}}`
}

func cycleBuildStuckGetItem(status, reason string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			Method:     "POST",
			RequestURI: "/",
			Operation:  "DynamoDB_20120810.GetItem",
			Body:       "ignore",
		},
		Response: awsutil.Response{StatusCode: 200, Body: buildItem(status, reason, "")},
	}
}

func cycleBuildStuck(body string) awsutil.Cycle {
	return awsutil.Cycle{
		Request:  awsutil.Request{Body: "ignore"},
		Response: awsutil.Response{StatusCode: 200, Body: body},
	}
}

var cycleBuildStuckDescribeStacks = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Body:       `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<StackName>convox-httpd</StackName>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
					</member>
				</Stacks>
			</DescribeStacksResult>
		</DescribeStacksResponse>`,
	},
}

var cycleBuildStuckPutItem = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.PutItem",
		Body:       "ignore",
	},
	Response: awsutil.Response{StatusCode: 200, Body: `{}`},
}

func cycleBuildStuckDescribeTasks(body string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			Method:     "POST",
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
			Body:       "ignore",
		},
		Response: awsutil.Response{StatusCode: 200, Body: body},
	}
}

func taskResponse(status, stopped, containerReason string) string {
	return `{"tasks":[{
		"taskArn": "arn:aws:ecs:us-test-1:901416387788:task/cluster-test/50b8de99",
		"lastStatus": "` + status + `",
		"stoppedReason": "` + stopped + `",
		"containers": [{"reason": "` + containerReason + `"}]
	}]}`
}

func taskLaunchResponse(launchType, status string) string {
	return `{"tasks":[{
		"taskArn": "` + buildStuckTaskArn + `",
		"taskDefinitionArn": "arn:aws:ecs:us-test-1:901416387788:task-definition/convox-build:1",
		"containerInstanceArn": "arn:aws:ecs:us-test-1:901416387788:container-instance/cluster-test/e126c67d",
		"launchType": "` + launchType + `",
		"lastStatus": "` + status + `",
		"containers": [{"name": "build"}]
	}]}`
}

func TestFailBuildIfNotTerminal(t *testing.T) {
	for _, c := range []struct {
		status   string
		terminal bool
	}{
		{"created", false},
		{"running", false},
		{"complete", true},
		{"failed", true},
	} {
		t.Run(c.status, func(t *testing.T) {
			cycles := []awsutil.Cycle{cycleBuildStuckGetItem(c.status, "")}
			if !c.terminal {
				cycles = append(cycles, cycleBuildStuckDescribeStacks, cycleBuildStuckPutItem)
			}

			p, bodies := stubProvider(t, cycles...)

			assert.Equal(t, c.terminal, p.failBuildIfNotTerminal("httpd", "B123", "dial tcp 127.0.0.1:5140: connection refused"))
			require.Len(t, *bodies, len(cycles))

			if c.terminal {
				return
			}

			put := (*bodies)[2]
			assert.Contains(t, put, `"status":{"S":"failed"}`)
			assert.Contains(t, put, `"reason":{"S":"dial tcp 127.0.0.1:5140: connection refused"}`)
		})
	}
}

func TestFailBuildIfNotTerminalKeepsExistingReason(t *testing.T) {
	p, bodies := stubProvider(t, cycleBuildStuckGetItem("running", ""), cycleBuildStuckDescribeStacks, cycleBuildStuckPutItem)

	assert.False(t, p.failBuildIfNotTerminal("httpd", "B123", ""))
	require.Len(t, *bodies, 3)
	assert.NotContains(t, (*bodies)[2], `"reason"`)
}

func TestTaskStopReason(t *testing.T) {
	arn := "arn:aws:ecs:us-test-1:901416387788:task/cluster-test/50b8de99"

	for _, c := range []struct {
		name     string
		response string
		reason   string
		stopped  bool
	}{
		{
			name:     "task and container reasons",
			response: taskResponse("STOPPED", "Task failed to start", "CannotStartContainerError: failed to initialize logging driver"),
			reason:   "Task failed to start: CannotStartContainerError: failed to initialize logging driver",
			stopped:  true,
		},
		{
			name:     "task reason only",
			response: taskResponse("STOPPED", "Essential container in task exited", ""),
			reason:   "Essential container in task exited",
			stopped:  true,
		},
		{
			name:     "container reason only",
			response: taskResponse("STOPPED", "", "CannotPullContainerError"),
			reason:   "CannotPullContainerError",
			stopped:  true,
		},
		{
			name:     "no containers",
			response: `{"tasks":[{"lastStatus":"STOPPED","stoppedReason":"placement failed"}]}`,
			reason:   "placement failed",
			stopped:  true,
		},
		{
			name:     "still running",
			response: taskResponse("RUNNING", "", ""),
			reason:   "",
			stopped:  false,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, _ := stubProvider(t, cycleBuildStuckDescribeTasks(c.response))

			reason, stopped := p.taskStopReason(arn)

			assert.Equal(t, c.reason, reason)
			assert.Equal(t, c.stopped, stopped)
		})
	}
}

func TestWaitForTask(t *testing.T) {
	arn := "arn:aws:ecs:us-test-1:901416387788:task/cluster-test/50b8de99"

	for _, c := range []struct {
		name     string
		statuses []string
		expect   string
	}{
		{"fargate start failure", []string{"PROVISIONING", "PENDING", "STOPPED"}, "STOPPED"},
		{"fargate healthy launch", []string{"PROVISIONING", "ACTIVATING", "RUNNING"}, "RUNNING"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cycles := []awsutil.Cycle{}
			for _, s := range c.statuses {
				cycles = append(cycles, cycleBuildStuckDescribeTasks(taskResponse(s, "", "")))
			}

			p, bodies := stubProvider(t, cycles...)

			status, err := p.waitForTask(arn, time.Now().Add(time.Minute))

			require.NoError(t, err)
			assert.Equal(t, c.expect, status)
			assert.Len(t, *bodies, len(c.statuses))
		})
	}
}

const buildStuckRackStack = `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
	<DescribeStacksResult>
		<Stacks>
			<member>
				<StackName>convox</StackName>
				<StackStatus>UPDATE_COMPLETE</StackStatus>
				<Parameters>
					<member>
						<ParameterKey>BuildMethod</ParameterKey>
						<ParameterValue>ec2</ParameterValue>
					</member>
				</Parameters>
			</member>
		</Stacks>
	</DescribeStacksResult>
</DescribeStacksResponse>`

const buildStuckCallerIdentity = `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
	<GetCallerIdentityResult>
		<Arn>arn:aws:iam::901416387788:user/test</Arn>
		<UserId>AIDATEST</UserId>
		<Account>901416387788</Account>
	</GetCallerIdentityResult>
</GetCallerIdentityResponse>`

func buildStuckStackResource(logical, physical string) string {
	return `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
	<ListStackResourcesResult>
		<StackResourceSummaries>
			<member>
				<LogicalResourceId>` + logical + `</LogicalResourceId>
				<PhysicalResourceId>` + physical + `</PhysicalResourceId>
				<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
			</member>
		</StackResourceSummaries>
	</ListStackResourcesResult>
</ListStackResourcesResponse>`
}

func cyclesBuildStuckLaunch() []awsutil.Cycle {
	return []awsutil.Cycle{
		cycleBuildStuck(buildStuckRackStack),
		cycleBuildStuck(buildStuckRackStack),
		cycleBuildStuck(buildStuckStackResource("ApiBuildTasks", "arn:aws:ecs:us-test-1:901416387788:task-definition/convox-build:1")),
		cycleBuildStuck(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>convox-settings</Name><KeyCount>0</KeyCount><IsTruncated>false</IsTruncated></ListBucketResult>`),
		cycleBuildStuck(buildStuckCallerIdentity),
		cycleBuildStuck(`{"authorizationData":[{"authorizationToken":"dXNlcjoxMjM0NQ==","proxyEndpoint":"https://901416387788.dkr.ecr.us-test-1.amazonaws.com"}]}`),
		cycleBuildStuck(buildStuckCallerIdentity),
		cycleBuildStuck(buildStuckStackResource("Registry", "convox-httpd-registry")),
		cycleBuildStuckDescribeStacks,
		cycleBuildStuck(buildStuckRackStack),
		cycleBuildStuck(taskLaunchResponse("EC2", "PENDING")),
		cycleBuildStuckGetItem("created", ""),
		cycleBuildStuckDescribeStacks,
		cycleBuildStuckPutItem,
	}
}

func TestRunBuildWatchesTaskStart(t *testing.T) {
	stopped := taskResponse("STOPPED", "Task failed to start", "CannotStartContainerError: failed to initialize logging driver")
	missing := cycleBuildStuckDescribeTasks(`{"tasks":[],"failures":[{"arn":"` + buildStuckTaskArn + `","reason":"MISSING"}]}`)

	for _, c := range []struct {
		name    string
		timeout time.Duration
		watcher []awsutil.Cycle
		reason  string
	}{
		{
			name:    "running",
			watcher: []awsutil.Cycle{cycleBuildStuckDescribeTasks(taskResponse("RUNNING", "", ""))},
		},
		{
			name: "stopped",
			watcher: []awsutil.Cycle{
				cycleBuildStuckDescribeTasks(stopped),
				cycleBuildStuckDescribeTasks(stopped),
				cycleBuildStuckGetItem("running", ""),
				cycleBuildStuckDescribeStacks,
				cycleBuildStuckPutItem,
			},
			reason: "Task failed to start: CannotStartContainerError: failed to initialize logging driver",
		},
		{
			name: "stopped without a reason",
			watcher: []awsutil.Cycle{
				cycleBuildStuckDescribeTasks(taskResponse("STOPPED", "", "")),
				cycleBuildStuckDescribeTasks(taskResponse("STOPPED", "", "")),
				cycleBuildStuckGetItem("running", ""),
				cycleBuildStuckDescribeStacks,
				cycleBuildStuckPutItem,
			},
			reason: "task stopped",
		},
		{
			name: "stopped after a terminal status",
			watcher: []awsutil.Cycle{
				cycleBuildStuckDescribeTasks(stopped),
				cycleBuildStuckDescribeTasks(stopped),
				cycleBuildStuckGetItem("failed", ""),
			},
		},
		{
			name: "describe errors, then stopped",
			watcher: []awsutil.Cycle{
				missing,
				missing,
				missing,
				cycleBuildStuckDescribeTasks(stopped),
				cycleBuildStuckDescribeTasks(stopped),
				cycleBuildStuckGetItem("running", ""),
				cycleBuildStuckDescribeStacks,
				cycleBuildStuckPutItem,
			},
			reason: "Task failed to start: CannotStartContainerError: failed to initialize logging driver",
		},
		{
			name:    "describe errors past the start timeout",
			timeout: 1500 * time.Millisecond,
			watcher: []awsutil.Cycle{missing, missing, missing},
		},
		{
			name:    "start timeout",
			timeout: 1500 * time.Millisecond,
			watcher: []awsutil.Cycle{
				cycleBuildStuckDescribeTasks(taskResponse("PENDING", "", "")),
				cycleBuildStuckDescribeTasks(taskResponse("PENDING", "", "")),
				cycleBuildStuck(`{}`),
				cycleBuildStuckGetItem("running", ""),
				cycleBuildStuckDescribeStacks,
				cycleBuildStuckPutItem,
			},
			reason: "task did not start within 60 minutes",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			launch := cyclesBuildStuckLaunch()
			inner := awsutil.NewHandler(append(launch, c.watcher...))

			var mu sync.Mutex
			var describedOnce, releaseOnce sync.Once
			bodies := []string{}
			described := make(chan struct{})
			release := make(chan struct{})

			p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.Header.Get("X-Amz-Target"), ".DescribeTasks") {
					describedOnce.Do(func() { close(described) })
					<-release
				}

				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %s", err)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(b))

				mu.Lock()
				defer mu.Unlock()

				bodies = append(bodies, string(b))
				inner.ServeHTTP(w, r)
			}))

			p.Cluster = "cluster-main"

			recorded := func() []string {
				mu.Lock()
				defer mu.Unlock()
				return append([]string{}, bodies...)
			}

			if c.timeout > 0 {
				timeout := taskStartTimeout
				taskStartTimeout = c.timeout
				t.Cleanup(func() { taskStartTimeout = timeout })
			}

			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				buildWatchers.Wait()
			})

			returned := make(chan error, 1)
			go func() {
				returned <- p.runBuild(&structs.Build{Id: "B123", App: "httpd", Tags: map[string]string{}}, "object:///httpd/source.tgz", structs.BuildCreateOptions{})
			}()

			select {
			case err := <-returned:
				require.NoError(t, err)
			case <-described:
				t.Fatal("runBuild described the task before returning")
			}

			launched := recorded()
			require.Len(t, launched, len(launch))
			assert.Contains(t, launched[len(launch)-1], `"status":{"S":"running"}`)

			select {
			case <-described:
			case <-time.After(10 * time.Second):
				t.Fatal("no watcher described the task")
			}

			releaseOnce.Do(func() { close(release) })
			buildWatchers.Wait()

			all := recorded()
			require.Len(t, all, len(launch)+len(c.watcher))

			if c.name == "start timeout" {
				assert.Equal(t, `{"cluster":"cluster-test","task":"`+buildStuckTaskArn+`"}`, all[len(launch)+2])
			}

			if c.reason == "" {
				return
			}

			put := all[len(all)-1]
			assert.Contains(t, put, `"status":{"S":"failed"}`)
			assert.Contains(t, put, `"reason":{"S":"`+c.reason+`"}`)
		})
	}
}

func TestBuildStopLine(t *testing.T) {
	task := map[string]string{"task": buildStuckTaskArn}

	for _, c := range []struct {
		name  string
		build structs.Build
		line  string
	}{
		{"stopped before running", structs.Build{Status: "failed", Reason: "CannotPullContainerError", Tags: task}, "build task stopped: CannotPullContainerError\n"},
		{"with logs", structs.Build{Status: "failed", Reason: "OutOfMemoryError", Logs: "object:///build/B123/logs", Tags: task}, ""},
		{"no task", structs.Build{Status: "failed", Reason: "not enough memory available to start process", Tags: map[string]string{}}, ""},
		{"no reason", structs.Build{Status: "failed", Tags: task}, ""},
		{"running", structs.Build{Status: "running", Reason: "CannotPullContainerError", Tags: task}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.line, buildStopLine(&c.build))
		})
	}
}

func TestBuildLogsFailedStartReason(t *testing.T) {
	p, bodies := stubProvider(t, cycleBuildStuckGetItem("failed", "CannotPullContainerError"))

	r, err := p.BuildLogs("httpd", "B123", structs.LogsOptions{})
	require.NoError(t, err)

	data, err := io.ReadAll(r)
	require.NoError(t, err)

	assert.Equal(t, "build task stopped: CannotPullContainerError\n", string(data))
	assert.Len(t, *bodies, 1)
}

func TestBuildLogsEC2TaskStopped(t *testing.T) {
	reason := "Task failed to start: CannotStartContainerError: failed to initialize logging driver"
	stopped := taskResponse("STOPPED", "Task failed to start", "CannotStartContainerError: failed to initialize logging driver")

	p, bodies := stubProvider(t,
		cycleBuildStuckGetItem("running", ""),
		cycleBuildStuckDescribeTasks(taskLaunchResponse("EC2", "PENDING")),
		cycleBuildStuckDescribeTasks(stopped),
		cycleBuildStuckDescribeTasks(stopped),
		cycleBuildStuckGetItem("running", ""),
		cycleBuildStuckDescribeStacks,
		cycleBuildStuckPutItem,
		cycleBuildStuckGetItem("failed", reason),
	)

	r, err := p.BuildLogs("httpd", "B123", structs.LogsOptions{})
	require.NoError(t, err)

	data, err := io.ReadAll(r)
	require.NoError(t, err)

	assert.Equal(t, "build task stopped: "+reason+"\n", string(data))
	require.Len(t, *bodies, 8)
	assert.Contains(t, (*bodies)[6], `"status":{"S":"failed"}`)
	assert.Contains(t, (*bodies)[6], `"reason":{"S":"`+reason+`"}`)
}

func TestBuildLogsFargate(t *testing.T) {
	startReason := "Task failed to start: CannotStartContainerError: failed to initialize logging driver"
	killReason := "Essential container in task exited: OutOfMemoryError: Container killed due to memory usage"

	notFound := awsutil.Response{StatusCode: 400, Body: `{"__type":"ResourceNotFoundException","message":"The specified log stream does not exist."}`}
	events := awsutil.Response{StatusCode: 200, Body: `{"events":[{"message":"Step 1/2 : FROM httpd"}],"nextForwardToken":"f/1"}`}
	drained := awsutil.Response{StatusCode: 200, Body: `{"events":[],"nextForwardToken":"f/1"}`}

	for _, c := range []struct {
		name   string
		builds []string
		tasks  []string
		logs   []awsutil.Response
		reason string
		stream string
	}{
		{
			name:   "stopped with no stream",
			builds: []string{buildItem("running", "", ""), buildItem("running", "", ""), buildItem("failed", startReason, "")},
			tasks: []string{
				taskLaunchResponse("FARGATE", "PROVISIONING"),
				taskResponse("STOPPED", "Task failed to start", "CannotStartContainerError: failed to initialize logging driver"),
				taskResponse("STOPPED", "Task failed to start", "CannotStartContainerError: failed to initialize logging driver"),
			},
			logs:   []awsutil.Response{notFound},
			reason: startReason,
			stream: "build task stopped: " + startReason + "\n",
		},
		{
			name:   "drained, build complete",
			builds: []string{buildItem("running", "", ""), buildItem("complete", "", ""), buildItem("complete", "", "stored build log")},
			tasks: []string{
				taskLaunchResponse("FARGATE", "RUNNING"),
				taskResponse("STOPPED", "Essential container in task exited", ""),
				taskResponse("STOPPED", "Essential container in task exited", ""),
			},
			logs:   []awsutil.Response{events, drained},
			stream: "Step 1/2 : FROM httpd\n",
		},
		{
			name:   "drained, task killed",
			builds: []string{buildItem("running", "", ""), buildItem("running", "", ""), buildItem("failed", killReason, "")},
			tasks: []string{
				taskLaunchResponse("FARGATE", "RUNNING"),
				taskResponse("STOPPED", "Essential container in task exited", "OutOfMemoryError: Container killed due to memory usage"),
				taskResponse("STOPPED", "Essential container in task exited", "OutOfMemoryError: Container killed due to memory usage"),
			},
			logs:   []awsutil.Response{events, drained},
			reason: killReason,
			stream: "Step 1/2 : FROM httpd\nbuild task stopped: " + killReason + "\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			ok := func(bodies ...string) []awsutil.Response {
				rs := []awsutil.Response{}
				for _, b := range bodies {
					rs = append(rs, awsutil.Response{StatusCode: 200, Body: b})
				}
				return rs
			}

			queues := map[string][]awsutil.Response{
				"DynamoDB_20120810.GetItem":                                 ok(c.builds...),
				"AmazonEC2ContainerServiceV20141113.DescribeTasks":          ok(c.tasks...),
				"AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition": ok(`{"taskDefinition":{"containerDefinitions":[{"name":"build","logConfiguration":{"logDriver":"awslogs","options":{"awslogs-group":"convox-build","awslogs-stream-prefix":"convox"}}}]}}`),
				"Logs_20140328.GetLogEvents":                                c.logs,
			}

			if c.reason != "" {
				queues[""] = []awsutil.Response{cycleBuildStuckDescribeStacks.Response}
				queues["DynamoDB_20120810.PutItem"] = ok(`{}`)
			}

			var mu sync.Mutex
			var put string
			unexpected := []string{}

			p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %s", err)
					return
				}

				op := r.Header.Get("X-Amz-Target")

				mu.Lock()
				defer mu.Unlock()

				if op == "DynamoDB_20120810.PutItem" {
					put = string(body)
				}

				q := queues[op]
				if len(q) == 0 {
					unexpected = append(unexpected, op)
					w.WriteHeader(404)
					return
				}

				if op != "Logs_20140328.GetLogEvents" || len(q) > 1 {
					queues[op] = q[1:]
				}

				w.WriteHeader(q[0].StatusCode)
				_, _ = io.WriteString(w, q[0].Body)
			}))

			r, err := p.BuildLogs("httpd", "B123", structs.LogsOptions{})
			require.NoError(t, err)

			read := make(chan string, 1)
			go func() {
				data, _ := io.ReadAll(r)
				read <- string(data)
			}()

			select {
			case stream := <-read:
				assert.Equal(t, c.stream, stream)
			case <-time.After(10 * time.Second):
				t.Fatal("the log stream did not end")
			}

			mu.Lock()
			defer mu.Unlock()

			assert.Empty(t, unexpected)

			for op, q := range queues {
				if op != "Logs_20140328.GetLogEvents" {
					assert.Empty(t, q, op)
				}
			}

			if c.reason != "" {
				assert.Contains(t, put, `"status":{"S":"failed"}`)
				assert.Contains(t, put, `"reason":{"S":"`+c.reason+`"}`)
			}
		})
	}
}
