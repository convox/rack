package aws_test

import (
	"fmt"
	"net/http/httptest"
	"os"

	"github.com/convox/rack/pkg/test/awsutil"
)

func stubDocker(cycles ...awsutil.Cycle) *httptest.Server {
	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)

	os.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://%s", s.URL[7:]))

	return s
}
