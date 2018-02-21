package aws_test

import (
	"bytes"
	"net/http/httptest"
	"os"

	"github.com/convox/logger"
	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/test/awsutil"
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

	os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	p := &aws.AWSProvider{
		Region:           "us-test-1",
		Endpoint:         s.URL,
		BuildCluster:     "cluster-test",
		Cluster:          "cluster-test",
		Development:      true,
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
