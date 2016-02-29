package controllers_test

import (
	"encoding/json"
	"net/url"
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
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),

		test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),

		test.DescribeServicesCycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		// query for every ECS task to get docker id, command, created
		test.ListECSContainersCycle(),

		// query every instance for one-off containers
		test.ListEmptyOneoffContainersCycle(),
		test.ListEmptyOneoffContainersCycle(),
		test.ListEmptyOneoffContainersCycle(),

		// query for every container to get CPU and Memory stats
		test.StatsCycle(),
	)
	defer docker.Close()

	v := url.Values{}
	v.Add("stats", "true")
	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", v)

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 1, len(resp))
		assert.Equal(t, 0.0974, resp[0].Memory)
	}
}

func TestGetProcessesWithDeployments(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")
	os.Setenv("TEST", "true")

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackCycle("convox-test-myapp-staging"),

		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),

		test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),

		test.DescribeServicesWithDeploymentsCycle("convox-test-cluster"),
		test.DescribeTaskDefinition3Cycle("convox-test-cluster"),
		test.DescribeTaskDefinition1Cycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		// query for every ECS task to get docker id, command, created
		test.ListECSContainersCycle(),

		// query every instance for one-off containers
		test.ListEmptyOneoffContainersCycle(),
		test.ListEmptyOneoffContainersCycle(),
		test.ListEmptyOneoffContainersCycle(),

		// query for every container to get CPU and Memory stats
		test.StatsCycle(),
	)
	defer docker.Close()

	v := url.Values{}
	v.Add("stats", "true")
	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", v)

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 2, len(resp))
		assert.Equal(t, "4932cce0897b", resp[0].Id)
		assert.Equal(t, 0.0974, resp[0].Memory)
		assert.Equal(t, "pending", resp[1].Id)
		assert.EqualValues(t, 0, resp[1].Memory)
	}
}

// func TestProcessShow(t *testing.T) {}

// func TestProcessStop(t *testing.T) {}

// func TestProcessRun(t *testing.T) {}

// func TestGetProcessesEmpty(t *testing.T) {}

// func TestGetProcessesFailure(t *testing.T) {}

// func TestGetProcessesWithDockerStats(t *testing.T) {}

// func TestGetProessesWithDockerStatsInDevelopment(t *testing.T) {}
