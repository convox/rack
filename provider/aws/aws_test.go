package aws_test

import (
	"net/http/httptest"
	"os"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/provider/aws"
)

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

	os.Setenv("AWS_ACCESS", "test")
	os.Setenv("AWS_SECRET", "test")
	os.Setenv("AWS_ENDPOINT", s.URL)
	os.Setenv("AWS_REGION", "us-test-1")
	os.Setenv("RACK", "convox")

	p := aws.NewProvider("us-test-1", s.URL, "test", "test", "")
	p.Cache = false

	return &AwsStub{p, s}
}
