package aws_test

import (
	"bytes"
	"net/http/httptest"

	"github.com/convox/logger"
	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/provider/aws"
)

func init() {
	logger.Output = &bytes.Buffer{}
}

type AwsStub struct {
	*aws.AWSProvider
	server *httptest.Server
}

func (a *AwsStub) Close() {
	a.server.Close()
}

// StubAwsProvider creates an httptest server with canned Request / Response
// cycles, and sets CurrentProvider to a new AWS provider that uses
// the test server as the endpoint
func StubAwsProvider(cycles ...awsutil.Cycle) *AwsStub {
	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)

	p := &aws.AWSProvider{
		Region:           "us-test-1",
		Endpoint:         s.URL,
		Access:           "test-access",
		Secret:           "test-secret",
		Token:            "test-token",
		BuildCluster:     "cluster-test",
		Cluster:          "cluster-test",
		Development:      true,
		DockerImageAPI:   "rack/web",
		DynamoBuilds:     "convox-builds",
		DynamoReleases:   "convox-releases",
		NotificationHost: "notifications.example.org",
		Password:         "password",
		Rack:             "convox",
		RegistryHost:     "registry.example.org",
		SettingsBucket:   "convox-settings",
		SkipCache:        true,
	}

	return &AwsStub{p, s}
}
