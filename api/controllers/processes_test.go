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
		test.DescribeAppStackCycle("myapp-staging"),
		test.DescribeAppStackCycle("myapp-staging"),
		test.ListTasksCycle("convox-test-cluster"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),
		test.DescribeContainerInstancesFilteredCycle("convox-test-cluster"),
		test.DescribeInstancesFilteredCycle(),
		test.DescribeAppStackResourcesCycle("myapp-staging"),
		test.DescribeServicesCycle("convox-test-cluster"),
		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),
	)
	defer aws.Close()

	docker := test.StubDocker(
		test.ListContainersCycle(),
		test.ListConvoxContainersCycle(),
		test.ListConvoxContainersCycle(),
		test.ListConvoxContainersCycle(),
		test.StatsCycle(),
	)
	defer docker.Close()

	// Note: there is a synchronization issue inside the Docker Stats fanout
	// So while the StatsCycle does work sometimes, the test bypasses stats for now
	v := url.Values{}
	v.Add("stats", "false")
	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", v)

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 1, len(resp))
		assert.Equal(t, 0.0, resp[0].Memory)
	}
}

func TestGetProcessesWithDeployments(t *testing.T) {
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
		test.DescribeAppStackResourcesCycle("myapp-staging"),
		test.DescribeServicesWithDeploymentsCycle("convox-test-cluster"),
		test.DescribeTaskDefinition3Cycle("convox-test-cluster"),
		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),
		test.DescribeTaskDefinition1Cycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		test.ListContainersCycle(),
		test.ListConvoxContainersCycle(),
		test.ListConvoxContainersCycle(),
		test.ListConvoxContainersCycle(),
		test.StatsCycle(),
	)
	defer docker.Close()

	v := url.Values{}
	v.Add("stats", "false")
	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", v)

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 2, len(resp))
		assert.Equal(t, "pending", resp[1].Id)
		//assert.Equal(t, "8dfafdbc3a40", resp[1].Id)
		//assert.Equal(t, "8dfafdbc3a40", resp[2].Id)
		//assert.Equal(t, "4932cce0897b", resp[3].Id)
		//assert.Equal(t, "pending", resp[4].Id)
	}
}

// func TestProcessShow(t *testing.T) {}

// func TestProcessStop(t *testing.T) {}

// func TestProcessRun(t *testing.T) {}

// func TestGetProcessesEmpty(t *testing.T) {}

// func TestGetProcessesFailure(t *testing.T) {}

// func TestGetProcessesWithDockerStats(t *testing.T) {}

// func TestGetProessesWithDockerStatsInDevelopment(t *testing.T) {}
