package controllers_test

import (
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
	os.Setenv("DYNAMO_RELEASES", "convox-releases")
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

// empty string for count should retain MainDesiredCount=1 in the stack update
func TestFormationScaleCountEmpty(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "256"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post count=2 should set MainDesiredCount=2 in the stack update
func TestFormationScaleCountTwo(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "2", "256"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{"2"}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post count=0 should set MainDesiredCount=0 in the stack update
func TestFormationScaleCountZero(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "0", "256"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{"0"}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}
