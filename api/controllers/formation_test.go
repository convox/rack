package controllers_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox-test")
	os.Setenv("DYNAMO_RELEASES", "convox-releases")
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

// empty string for count should retain MainDesiredCount=1 and MainMemory=256 in the stack update
func TestFormationScaleEmpty(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		Capacity: structs.Capacity{},
	}

	// setup expectations on current provider
	models.TestProvider.On("CapacityGet").Return(models.TestProvider.Capacity, nil)

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

// post count=foo should 403
func TestFormationScaleCountInvalid(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "0", "256"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{"foo"}, "memory": []string{""}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"error":"count must be numeric"}`, body)
}

// post count=2 should set MainDesiredCount=2 in the stack update
func TestFormationScaleCount2(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		Capacity: structs.Capacity{},
	}

	// setup expectations on current provider
	models.TestProvider.On("CapacityGet").Return(models.TestProvider.Capacity, nil)

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
func TestFormationScaleCount0(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		Capacity: structs.Capacity{},
	}

	// setup expectations on current provider
	models.TestProvider.On("CapacityGet").Return(models.TestProvider.Capacity, nil)

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

// post memory=foo should 403
func TestFormationScaleMemoryInvalid(t *testing.T) {
	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "256"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{"foo"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"error":"memory must be numeric"}`, body)
}

// post memory=0 should retain MainMemory=256 in the stack update
func TestFormationScaleMemory0(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		Capacity: structs.Capacity{},
	}

	// setup expectations on current provider
	models.TestProvider.On("CapacityGet").Return(models.TestProvider.Capacity, nil)

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "256"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{"0"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post memory=512 should set MainMemory=512 in the stack update
func TestFormationScaleMemory512(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		Capacity: structs.Capacity{
			InstanceMemory: 2048,
		},
	}

	// setup expectations on current provider
	models.TestProvider.On("CapacityGet").Return(models.TestProvider.Capacity, nil)

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "512"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{"512"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"success":true}`, body)
}

// post memory=2048 should error
func TestFormationScaleMemory2048(t *testing.T) {
	// set current provider
	models.TestProvider = &provider.TestProvider{
		Capacity: structs.Capacity{
			InstanceMemory: 1024,
		},
	}

	models.TestProvider.On("CapacityGet").Return(models.TestProvider.Capacity, nil)

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-application"),
		test.DescribeAppStackCycle("convox-test-application"),
		test.GetItemAppReleaseCycle("convox-test-application"),
		test.UpdateAppStackCycle("convox-test-application", "1", "512"),
	)
	defer aws.Close()

	val := url.Values{"count": []string{""}, "memory": []string{"2048"}}
	body := test.HTTPBody("POST", "http://convox/apps/application/formation/main", val)

	assert.Equal(t, `{"error":"requested memory 2048 greater than instance size 1024"}`, body)
}
