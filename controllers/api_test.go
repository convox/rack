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

func stubDescribeStack(stackName string) (s *httptest.Server) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStackCycle(stackName),
	})
	s = httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	return
}

func TestNoPassword(t *testing.T) {
	server := stubDescribeStack("convox-test")
	defer server.Close()
	defer os.Setenv("RACK", os.Getenv("RACK"))

	os.Setenv("RACK", "convox-test")

	assert.HTTPSuccess(t, controllers.HandlerFunc, "GET", "http://convox/system", nil)
}

func TestBasicAuth(t *testing.T) {
	assert := assert.New(t)
	server := stubDescribeStack("convox-test")
	defer server.Close()
	defer os.Setenv("PASSWORD", os.Getenv("PASSWORD"))
	defer os.Setenv("RACK", os.Getenv("RACK"))

	os.Setenv("PASSWORD", "keymaster")
	os.Setenv("RACK", "convox-test")

	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.HandlerFunc(w, req)

	if !assert.Equal(401, w.Code) {
		return
	}

	w = httptest.NewRecorder()
	req.SetBasicAuth("", "keymaster")
	controllers.HandlerFunc(w, req)

	assert.Equal(200, w.Code)
}
