package controllers_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox-test")
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

// func TestAppList(t *testing.T) {
//   aws := test.StubAws(test.DescribeStackCycleWithoutQuery("convox-test-bar"))
//   defer aws.Close()

//   body := test.HTTPBody("GET", "http://convox/apps", nil, nil)

//   var resp models.Apps
//   err := json.Unmarshal([]byte(body), &resp)

//   if assert.NoError(t, err) {
//     assert.Equal(t, "bar", resp[0].Name)
//     assert.Equal(t, "running", resp[0].Status)
//   }
// }

func TestAppGet(t *testing.T) {
	models.Test(t, func() {
		app := &structs.App{
			Name:    "myapp",
			Release: "R1234",
			Status:  "running",
		}

		models.TestProvider.On("AppGet", "myapp").Return(app, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"name\":\"myapp\",\"release\":\"R1234\",\"status\":\"running\"}")
		}
	})
}

func TestAppGetWithAppNotFound(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("AppGet", "myapp").Return(nil, errorNotFound(fmt.Sprintf("no such app: myapp")))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertJSON(t, "{\"error\":\"no such app: myapp\"}")
		}
	})
}

// Test the primary path: creating an app on a `convox` rack
// Return to testing against a `convox-test` rack afterwards
// func TestAppCreate(t *testing.T) {
//   r := os.Getenv("RACK")
//   os.Setenv("RACK", "convox")
//   defer os.Setenv("RACK", r)

//   aws := test.StubAws(
//     test.DescribeStackNotFound("application"),
//     test.CreateAppStackCycle("convox-application"),
//     test.DescribeAppStackCycle("convox-application"),
//   )
//   defer aws.Close()

//   val := url.Values{"name": []string{"application"}}
//   body := test.HTTPBody("POST", "http://convox/apps", val, nil)

//   if assert.NotEqual(t, "", body) {
//     var resp map[string]string
//     err := json.Unmarshal([]byte(body), &resp)

//     if assert.NoError(t, err) {
//       assert.Equal(t, "application", resp["name"])
//       assert.Equal(t, "running", resp["status"])
//     }
//   }
// }

// func TestAppCreateWithAlreadyExists(t *testing.T) {
//   aws := test.StubAws(
//     test.DescribeStackNotFound("application"),
//     test.CreateAppStackExistsCycle("convox-test-application"),
//     test.DescribeAppStackCycle("convox-test-application"),
//   )
//   defer aws.Close()

//   val := url.Values{"name": []string{"application"}}
//   body := test.AssertStatus(t, 403, "POST", "http://convox/apps", val, nil)
//   assert.Equal(t, "{\"error\":\"there is already an app named application (running)\"}", body)
// }

// func TestAppCreateWithAlreadyExistsUnbound(t *testing.T) {
//   aws := test.StubAws(
//     test.DescribeAppStackCycle("application"),
//   )
//   defer aws.Close()

//   val := url.Values{"name": []string{"application"}}
//   body := test.AssertStatus(t, 403, "POST", "http://convox/apps", val, nil)
//   assert.Equal(t, "{\"error\":\"there is already a legacy app named application (running). We recommend you delete this app and create it again.\"}", body)
// }

// func TestAppCreateWithRackName(t *testing.T) {
//   aws := test.StubAws(
//     test.DescribeAppStackCycle("foobar"),
//   )
//   defer aws.Close()

//   val := url.Values{"name": []string{"convox-test"}}
//   body := test.AssertStatus(t, 403, "POST", "http://convox/apps", val, nil)
//   assert.Equal(t, "{\"error\":\"application name cannot match rack name (convox-test). Please choose a different name for your app.\"}", body)
// }

// TODO: test bucket cleanup. this is handled via goroutines.
/* NOTE: the S3 stuff fucks up b.c the client ries to prepend the
bucket name to the ephermeral host, so you get `app-XXX.127.0.0.1`
*/
// func TestAppDelete(t *testing.T) {
//   aws := test.StubAws(
//     test.DescribeAppStackCycle("convox-test-bar"),
//     test.DeleteStackCycle("convox-test-bar"),
//   )
//   defer aws.Close()

//   // setup expectations on current provider
//   models.TestProvider.On("AppDelete", "bar").Return(nil)

//   body := test.HTTPBody("DELETE", "http://convox/apps/bar", nil, nil)

//   var resp map[string]bool
//   err := json.Unmarshal([]byte(body), &resp)

//   if assert.NoError(t, err) {
//     assert.Equal(t, true, resp["success"])
//   }
// }

// func TestAppDeleteWithAppNotFound(t *testing.T) {
//   aws := test.StubAws(
//     test.DescribeStackNotFound("convox-test-bar"),
//     test.DescribeStackNotFound("bar"),
//   )
//   defer aws.Close()

//   test.AssertStatus(t, 404, "DELETE", "http://convox/apps/bar", nil, nil)
// }

func TestAppLogs(t *testing.T) {

}
