package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/kernel/awsutil"
	"github.com/convox/kernel/controllers"
)

func TestAppList(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStackCycleWithoutQuery("bar"),
	})
	s := httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	defer s.Close()

	body := assert.HTTPBody(controllers.HandlerFunc, "GET", "http://convox/apps", nil)

	var resp []map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp[0]["name"])
		assert.Equal(t, "running", resp[0]["status"])
	}
}

func TestAppShowFound(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStackCycleWithQuery("bar"),
	})
	s := httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	defer s.Close()

	body := assert.HTTPBody(controllers.HandlerFunc, "GET", "http://convox/apps/bar", nil)

	var resp map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp["name"])
		assert.Equal(t, "running", resp["status"])
	}
}

func TestAppShowWithNoApp(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStackEmptyResponse("bar"),
	})
	s := httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	defer s.Close()

	req, _ := http.NewRequest("GET", "http://convox/apps/bar", nil)
	w := httptest.NewRecorder()
	controllers.HandlerFunc(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestAppCreate(t *testing.T) {

}

func TestAppCreateWithAlreadyExists(t *testing.T) {

}

func TestAppDelete(t *testing.T) {

}

func TestAppDeleteWithNoApp(t *testing.T) {

}

func TestAppLogs(t *testing.T) {

}
