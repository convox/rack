package controllers_test

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/test"
)

func init() {
	os.Setenv("RACK", "convox-test")
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

func TestAppList(t *testing.T) {
	aws := test.StubAws(test.DescribeStackCycleWithoutQuery("convox-test-bar"))
	defer aws.Close()

	body := test.HTTPBody("GET", "http://convox/apps", nil)

	var resp []map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp[0]["name"])
		assert.Equal(t, "running", resp[0]["status"])
	}
}

func TestAppShow(t *testing.T) {
	aws := test.StubAws(
		test.DescribeStackNotFound("convox-test-bar"),
		test.DescribeAppStackCycle("bar"),
	)
	defer aws.Close()

	body := test.HTTPBody("GET", "http://convox/apps/bar", nil)

	var resp map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp["name"])
		assert.Equal(t, "running", resp["status"])
	}
}

func TestAppShowWithAppNotFound(t *testing.T) {
	aws := test.StubAws(
		test.DescribeStackNotFound("convox-test-bar"),
		test.DescribeStackNotFound("bar"),
	)
	defer aws.Close()

	test.AssertStatus(t, 404, "GET", "http://convox/apps/bar", nil)
}

func TestAppCreate(t *testing.T) {
	aws := test.StubAws(
		test.DescribeStackNotFound("application"),
		test.CreateAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
	)
	defer aws.Close()

	val := url.Values{"name": []string{"application"}}
	body := test.HTTPBody("POST", "http://convox/apps", val)

	if assert.NotEqual(t, "", body) {
		var resp map[string]string
		err := json.Unmarshal([]byte(body), &resp)

		if assert.Nil(t, err) {
			assert.Equal(t, "application", resp["name"])
			assert.Equal(t, "running", resp["status"])
		}
	}
}

func TestAppCreateWithAlreadyExistsUnbound(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("application"),
		test.CreateAppStackExistsCycle("application"),
		test.DescribeStackNotFound("convox-test-application"),
		test.DescribeAppStackCycle("application"),
	)
	defer aws.Close()

	val := url.Values{"name": []string{"application"}}
	test.AssertStatus(t, 403, "POST", "http://convox/apps", val)
}

func TestAppCreateWithAlreadyExistsBound(t *testing.T) {
	aws := test.StubAws(
		test.DescribeStackNotFound("application"),
		test.CreateAppStackExistsCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
	)
	defer aws.Close()

	val := url.Values{"name": []string{"application"}}
	test.AssertStatus(t, 403, "POST", "http://convox/apps", val)
}

// TODO: test bucket cleanup. this is handled via goroutines.
/* NOTE: the S3 stuff fucks up b.c the client ries to prepend the
bucket name to the ephermeral host, so you get `app-XXX.127.0.0.1`
*/
func TestAppDelete(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-bar"),
		test.DeleteStackCycle("convox-test-bar"),
	)
	defer aws.Close()

	body := test.HTTPBody("DELETE", "http://convox/apps/bar", nil)

	var resp map[string]bool
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, true, resp["success"])
	}
}

func TestAppDeleteWithAppNotFound(t *testing.T) {
	aws := test.StubAws(
		test.DescribeStackNotFound("convox-test-bar"),
		test.DescribeStackNotFound("bar"),
	)
	defer aws.Close()

	test.AssertStatus(t, 404, "DELETE", "http://convox/apps/bar", nil)
}

func TestAppLogs(t *testing.T) {

}
