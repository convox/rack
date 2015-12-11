package controllers_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func init() {
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

func TestProcessesList(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")
	os.Setenv("TEST", "true")

	aws := test.StubAws(
		test.DescribeAppStackCycle("myapp-staging"),
		test.DescribeAppStackCycle("myapp-staging"),
		test.ListTasksCycle("convox-test-cluster"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),
		test.DescribeContainerInstancesFilteredCycle("convox-test-cluster"),
		test.DescribeInstancesFilteredCycle(),
		test.ListServicesCycle("convox-test-cluster"),
		test.DescribeServicesCycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		test.ListContainersCycle(),
		test.StatsCycle(),
	)
	defer docker.Close()

	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", nil)

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 1, len(resp))
	}
}

// func TestProcessShow(t *testing.T) {}

// func TestProcessStop(t *testing.T) {}

// func TestProcessRun(t *testing.T) {}

// func TestGetProcessesEmpty(t *testing.T) {}

// func TestGetProcessesFailure(t *testing.T) {}

// func TestGetProcessesWithDeployments(t *testing.T) {}

// func TestGetProcessesWithDockerStats(t *testing.T) {}

// func TestGetProessesWithDockerStatsInDevelopment(t *testing.T) {}
