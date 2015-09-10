package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/kernel/awsutil"
	"github.com/convox/kernel/controllers"
)

/*
func TestNoAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	if w.Code != 301 {
		t.Errorf("expected status code of %d, got %d", 301, w.Code)
		return
	}
}
*/

func stubSystem() {
	s := httptest.NewServer(awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "",
				Body:       `{"cluster":"convox"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskArns":["arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58"]}`,
			},
		},
	}))
	defer s.Close()

	aws.DefaultConfig.Region = "test"
	os.Setenv("AWS_REGION", "test")
	os.Setenv("CLUSTER", "convox")
	os.Setenv("DYNAMO_RELEASES", "releases")
	os.Setenv("TEST_DOCKER_HOST", s.URL)
}

func TestBasicAuth(t *testing.T) {
	defer func(p string) {
		os.Setenv("PASSWORD", p)
	}(os.Getenv("PASSWORD"))

	stubSystem()

	os.Setenv("PASSWORD", "keymaster")
	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	if w.Code != 401 {
		t.Errorf("expected status code of %d, got %d", 401, w.Code)
		return
	}

	w = httptest.NewRecorder()
	req.SetBasicAuth("", "keymaster")
	controllers.SingleRequest(w, req)

	if w.Code != 200 {
		t.Errorf("expected status code of %d, got %d", 200, w.Code)
		return
	}
}
