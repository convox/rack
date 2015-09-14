package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/controllers"
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
	aws := stubAws(DescribeAppStackCycle("bar"))
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
	aws := stubAws(
		CreateAppStackCycle("application"),
		DescribeAppStackCycle("application"),
	)
	defer aws.Close()

	val := url.Values{"name": []string{"application"}}
	postBody := strings.NewReader(val.Encode())
	req, _ := http.NewRequest("POST", "http://convox/apps", postBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	controllers.HandlerFunc(w, req)

	if assert.Equal(t, 200, w.Code) {
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)

		if assert.Nil(t, err) {
			assert.Equal(t, "application", resp["name"])
			assert.Equal(t, "running", resp["status"])
		}
	}
}

func TestAppCreateWithAlreadyExists(t *testing.T) {
	aws := stubAws(
		CreateAppStackExistsCycle("application"),
		DescribeAppStackCycle("application"),
	)
	defer aws.Close()

	val := url.Values{"name": []string{"application"}}
	postBody := strings.NewReader(val.Encode())
	req, _ := http.NewRequest("POST", "http://convox/apps", postBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	controllers.HandlerFunc(w, req)

	assert.Equal(t, 403, w.Code)
}

// TODO: test bucket cleanup. this is handled via goroutines.
/* NOTE: the S3 stuff fucks up b.c the client ries to prepend the
bucket name to the ephermeral host, so you get `app-XXX.127.0.0.1`
*/
func TestAppDelete(t *testing.T) {
	aws := stubAws(
		DescribeAppStackCycle("bar"),
		DeleteStackCycle("bar"),
	)
	defer aws.Close()

	body := assert.HTTPBody(controllers.HandlerFunc, "DELETE", "http://convox/apps/bar", nil)

	var resp map[string]bool
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, true, resp["success"])
	}
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
