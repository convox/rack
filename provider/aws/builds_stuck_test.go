package aws

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubProvider(t *testing.T, cycles ...awsutil.Cycle) (*Provider, *[]string) {
	t.Helper()
	t.Setenv("PROVIDER", "test")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-test-1")
	t.Setenv("AWS_SESSION_TOKEN", "")

	inner := awsutil.NewHandler(cycles)
	bodies := []string{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %s", err)
			return
		}

		bodies = append(bodies, string(b))
		r.Body = io.NopCloser(bytes.NewReader(b))

		inner.ServeHTTP(w, r)
	}))

	t.Cleanup(srv.Close)

	p := &Provider{
		BuildCluster: "cluster-test",
		Cluster:      "cluster-test",
		DynamoBuilds: "convox-builds",
		Endpoint:     srv.URL,
		Rack:         "convox",
		Region:       "us-test-1",
		SkipCache:    true,
	}

	return p, &bodies
}

func cycleBuildStuckGetItem(status string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			Method:     "POST",
			RequestURI: "/",
			Operation:  "DynamoDB_20120810.GetItem",
			Body:       "ignore",
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: `{"Item":{
				"id": {"S": "B123"},
				"app": {"S": "httpd"},
				"created": {"S": "20160404.143416.178278576"},
				"status": {"S": "` + status + `"}
			}}`,
		},
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
			cycles := []awsutil.Cycle{cycleBuildStuckGetItem(c.status)}
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
	p, bodies := stubProvider(t, cycleBuildStuckGetItem("running"), cycleBuildStuckDescribeStacks, cycleBuildStuckPutItem)

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

			status, err := p.waitForTask(arn)

			require.NoError(t, err)
			assert.Equal(t, c.expect, status)
			assert.Len(t, *bodies, len(c.statuses))
		})
	}
}
