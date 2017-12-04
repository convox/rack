package aws_test

import (
	"net/http/httptest"
	"os"

	"github.com/convox/rack/test/awsutil"
)

func stubDocker(cycles ...awsutil.Cycle) *httptest.Server {
	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)

	os.Setenv("DOCKER_HOST", s.URL[7:])

	return s
}
