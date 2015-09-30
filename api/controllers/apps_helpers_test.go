package controllers_test

import (
	"net/http/httptest"

	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/convox/rack/api/awsutil"
)

func init() {
	region := "test"
	defaults.DefaultConfig.Region = &region
}

/*
Create a test server that mocks an AWS request/response cycle,
suitable for a single test

Example:
		s := stubAws(DescribeStackCycleWithoutQuery("bar"))
		defer s.Close()
*/
func stubAws(cycles ...awsutil.Cycle) (s *httptest.Server) {
	handler := awsutil.NewHandler(cycles)
	s = httptest.NewServer(handler)
	defaults.DefaultConfig.Endpoint = &s.URL
	return s
}
