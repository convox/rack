package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/kernel/controllers"
)

func TestAppList(t *testing.T) {
	aws := stubAws(DescribeStackCycleWithoutQuery("bar"))
	defer aws.Close()

	body := assert.HTTPBody(controllers.HandlerFunc, "GET", "http://convox/apps", nil)

	var resp []map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp[0]["name"])
		assert.Equal(t, "running", resp[0]["status"])
	}
}

func TestAppShow(t *testing.T) {
	aws := stubAws(DescribeStackCycleWithQuery("bar"))
	defer aws.Close()

	body := assert.HTTPBody(controllers.HandlerFunc, "GET", "http://convox/apps/bar", nil)

	var resp map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp["name"])
		assert.Equal(t, "running", resp["status"])
	}
}

func TestAppShowWithAppNotFound(t *testing.T) {
	aws := stubAws(DescribeStackNotFound("bar"))
	defer aws.Close()

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

func TestAppDeleteWithAppNotFound(t *testing.T) {
	aws := stubAws(DescribeStackNotFound("bar"))
	defer aws.Close()

	req, _ := http.NewRequest("DELETE", "http://convox/apps/bar", nil)
	w := httptest.NewRecorder()
	controllers.HandlerFunc(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestAppLogs(t *testing.T) {

}
