package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/kernel/awsutil"
	"github.com/convox/kernel/controllers"
)

func stubSystem() (s *httptest.Server) {
	os.Setenv("RACK", "convox-test")
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStackCycle("convox-test"),
	})
	s = httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	return
}

func TestNoPassword(t *testing.T) {
	server := stubSystem()
	defer server.Close()

	assert.HTTPSuccess(t, controllers.SingleRequest, "GET", "http://convox/system", nil)
}

func TestBasicAuth(t *testing.T) {
	assert := assert.New(t)
	server := stubSystem()
	defer server.Close()
	defer os.Setenv("PASSWORD", os.Getenv("PASSWORD"))

	os.Setenv("PASSWORD", "keymaster")
	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	if !assert.Equal(401, w.Code) {
		return
	}

	w = httptest.NewRecorder()
	req.SetBasicAuth("", "keymaster")
	controllers.SingleRequest(w, req)

	assert.Equal(200, w.Code)
}
