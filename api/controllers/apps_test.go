package controllers_test

import (
	"encoding/json"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
)

func init() {
	test.HandlerFunc = controllers.HandlerFunc
}

func TestAppList(t *testing.T) {
	provider.TestProvider.Apps = structs.Apps{
		structs.App{Name: "bar", Status: "running"},
	}

	body := test.HTTPBody("GET", "http://convox/apps", nil)

	var resp []map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp[0]["name"])
		assert.Equal(t, "running", resp[0]["status"])
	}
}

func TestAppShow(t *testing.T) {
	provider.TestProvider.App = &structs.App{
		Name:   "bar",
		Status: "running",
	}

	body := test.HTTPBody("GET", "http://convox/apps/bar", nil)

	var resp map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp["name"])
		assert.Equal(t, "running", resp["status"])
	}
}

// FIXME implement in provider test

// func TestAppShowWithAppNotFound(t *testing.T) {
//   aws := test.StubAws(test.DescribeStackNotFound("bar"))
//   defer aws.Close()

//   test.AssertStatus(t, 404, "GET", "http://convox/apps/bar", nil)
// }

// FIXME implement in provider test

// func TestAppCreate(t *testing.T) {
//   aws := test.StubAws(
//     test.CreateAppStackCycle("application"),
//     test.DescribeAppStackCycle("application"),
//   )
//   defer aws.Close()

//   val := url.Values{"name": []string{"application"}}
//   body := test.HTTPBody("POST", "http://convox/apps", val)

//   if assert.NotEqual(t, "", body) {
//     var resp map[string]string
//     err := json.Unmarshal([]byte(body), &resp)

//     if assert.Nil(t, err) {
//       assert.Equal(t, "application", resp["name"])
//       assert.Equal(t, "running", resp["status"])
//     }
//   }
// }

// FIXME implement in provider test

// func TestAppCreateWithAlreadyExists(t *testing.T) {
//   aws := test.StubAws(
//     test.CreateAppStackExistsCycle("application"),
//     test.DescribeAppStackCycle("application"),
//   )
//   defer aws.Close()

//   val := url.Values{"name": []string{"application"}}
//   test.AssertStatus(t, 403, "POST", "http://convox/apps", val)
// }

// TODO: test bucket cleanup. this is handled via goroutines.
/* NOTE: the S3 stuff fucks up b.c the client ries to prepend the
bucket name to the ephermeral host, so you get `app-XXX.127.0.0.1`
*/
func TestAppDelete(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("bar"),
		test.DeleteStackCycle("bar"),
	)
	defer aws.Close()

	body := test.HTTPBody("DELETE", "http://convox/apps/bar", nil)

	var resp map[string]bool
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, true, resp["success"])
	}
}

// FIXME needs to be moved into provider test

// func TestAppDeleteWithAppNotFound(t *testing.T) {
//   aws := test.StubAws(test.DescribeStackNotFound("bar"))
//   defer aws.Close()

//   test.AssertStatus(t, 404, "DELETE", "http://convox/apps/bar", nil)
// }

// FIXME implement

func TestAppLogs(t *testing.T) {

}
