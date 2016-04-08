package aws_test

import (
	"net/http/httptest"
	"os"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/provider/aws"
)

// StubAwsProvider creates an httptest server with canned Request / Response
// cycles, and sets CurrentProvider to a new AWS provider that uses
// the test server as the endpoint
func StubAwsProvider(cycles ...awsutil.Cycle) (s *httptest.Server) {
	handler := awsutil.NewHandler(cycles)
	s = httptest.NewServer(handler)

	os.Setenv("AWS_ACCESS", "test")
	os.Setenv("AWS_SECRET", "test")
	os.Setenv("AWS_ENDPOINT", s.URL)
	os.Setenv("AWS_REGION", "test")

	p, err := aws.NewProvider("test", "test", "test", s.URL)

	if err != nil {
		panic(err)
	}

	provider.CurrentProvider = p

	return s
}
