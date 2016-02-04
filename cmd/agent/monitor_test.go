package main

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger"
	"github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestNewMonitor(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/info",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"Driver": "devicemapper", "KernelVersion": "4.1.13-19.31.amzn1.x86_64", "ServerVersion": "1.9.1"}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/json",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `[{"Id": "8dfafdbc3a40", "Image": "amazon/amazon-ecs-agent:latest"}]`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/8dfafdbc3a40/json",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"Id": "8dfafdbc3a4006a3b513bc9d639eee123ad78ca3616b921167cd74b20e25ed39", "Image": "46e05d1109686168630d3ba35d8889bd0e9caafcaeb3004d2bfbc47e7c5d35d2"}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/meta-data/instance-id",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `i-05c7e6b6fcc83ae8a`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/meta-data/ami-id",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `ami-cb2305a1`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/meta-data/placement/availability-zone",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `us-east-1c`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/meta-data/instance-id",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `i-05c7e6b6fcc83ae8a`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/meta-data/instance-type",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `r3.large`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/meta-data/placement/availability-zone",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `us-east-1c`,
			},
		},
	})
	s := httptest.NewServer(handler)

	os.Setenv("DOCKER_HOST", s.URL)
	os.Setenv("EC2_METADATA_ENDPOINT", s.URL)

	monitor := NewMonitor()

	assert.EqualValues(t,
		&Monitor{
			client: monitor.client,

			envs: make(map[string]map[string]string),

			agentId:    "unknown",
			agentImage: "convox/agent:dev",

			amiId:        "ami-cb2305a1",
			az:           "us-east-1c",
			instanceId:   "i-05c7e6b6fcc83ae8a",
			instanceType: "r3.large",
			region:       "us-east-1",

			dockerDriver:        "devicemapper",
			dockerServerVersion: "1.9.1",
			ecsAgentImage:       "46e05d110968",
			kernelVersion:       "4.1.13-19.31.amzn1.x86_64",

			lines:   make(map[string][][]byte),
			loggers: make(map[string]logger.Logger),
		},
		monitor,
	)
}
