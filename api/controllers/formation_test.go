package controllers_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
)

func init() {
	os.Setenv("RACK", "convox-test")
	os.Setenv("DYNAMO_RELEASES", "convox-releases")
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

// empty string for count should retain MainDesiredCount=1 and MainMemory=128 in the stack update
func TestFormationScaleEmpty(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "128"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post count=foo should 403
func TestFormationScaleCountInvalid(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "0", "128"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{"foo"}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"error":"count must be numeric"}`, body)
}

// post count=2 should set MainDesiredCount=2 in the stack update
func TestFormationScaleCount2(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "2", "128"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{"2"}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post count=0 should set MainDesiredCount=0 in the stack update
func TestFormationScaleCount0(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "0", "128"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{"0"}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post memory=foo should 403
func TestFormationScaleMemoryInvalid(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "128"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{"foo"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"error":"memory must be numeric"}`, body)
}

// post memory=0 should retain MainMemory=128 in the stack update
func TestFormationScaleMemory0(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "128"),
	)
	defer aws.Close()

	provider.TestProvider.Capacity = structs.Capacity{
		InstanceMemory: 1024,
	}

	val := url.Values{"count": []string{""}, "memory": []string{"0"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post memory=512 should set MainMemory=512 in the stack update
func TestFormationScaleMemory512(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "512"),
	)
	defer aws.Close()

	provider.TestProvider.Capacity = structs.Capacity{
		InstanceMemory: 1024,
	}

	val := url.Values{"count": []string{""}, "memory": []string{"512"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post memory=2048 should error
func TestFormationScaleMemory2048(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "512"),
	)
	defer aws.Close()

	provider.TestProvider.Capacity = structs.Capacity{
		InstanceMemory: 1024,
	}

	val := url.Values{"count": []string{""}, "memory": []string{"2048"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"error":"requested memory 2048 greater than instance size 1024"}`, body)
}
