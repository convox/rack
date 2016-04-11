package controllers_test

import (
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

func TestProcessesList(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")

	// set current provider
	testProvider := &provider.TestProviderRunner{
		Instances: []structs.Instance{
			structs.Instance{},
			structs.Instance{},
			structs.Instance{},
		},
	}
	provider.CurrentProvider = testProvider

	defer func() {
		//TODO: remove: as we arent updating all tests we need tos et current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	// setup expectations on current provider
	testProvider.On("InstanceList").Return(testProvider.Instances, nil)

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),

		test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.ListTasksOneoffEmptyCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),

		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
		test.DescribeServicesCycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		// query for every ECS task to get docker id, command, created
		test.ListECSContainersCycle(),

		// query every instance for one-off containers
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersEmptyCycle(),

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
	// set current provider
	testProvider := &provider.TestProviderRunner{
		Instances: []structs.Instance{
			structs.Instance{},
			structs.Instance{},
			structs.Instance{},
		},
	}
	provider.CurrentProvider = testProvider

	defer func() {
		//TODO: remove: as we arent updating all tests we need tos et current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	// setup expectations on current provider
	testProvider.On("InstanceList").Return(testProvider.Instances, nil)

	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),

		test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.ListTasksOneoffEmptyCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),

		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
		test.DescribeServicesWithDeploymentsCycle("convox-test-cluster"),
		test.DescribeTaskDefinition3Cycle("convox-test-cluster"),
		test.DescribeTaskDefinition1Cycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		// query for every ECS task to get docker id, command, created
		test.ListECSContainersCycle(),

		// query every instance for one-off containers
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersEmptyCycle(),

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
		assert.Equal(t, "8dfafdbc3a40", resp[0].Id)
		assert.Equal(t, 0.0974, resp[0].Memory)
		assert.Equal(t, "pending", resp[1].Id)
		assert.EqualValues(t, 0, resp[1].Memory)
	}
}

func TestProcessesListWithAttached(t *testing.T) {
	// set current provider
	testProvider := &provider.TestProviderRunner{
		Instances: []structs.Instance{
			structs.Instance{},
			structs.Instance{},
			structs.Instance{},
		},
	}
	provider.CurrentProvider = testProvider

	defer func() {
		//TODO: remove: as we arent updating all tests we need tos et current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	// setup expectations on current provider
	testProvider.On("InstanceList").Return(testProvider.Instances, nil)

	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),

		test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.ListTasksOneoffEmptyCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),

		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
		test.DescribeServicesCycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		// query for every ECS task to get docker id, command, created
		test.ListECSContainersCycle(),

		// query every instance for one-off containers
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersCycle("79bd711b1756"),
		test.InspectCycle("79bd711b1756"),
		test.ListOneoffContainersEmptyCycle(),
	)
	defer docker.Close()

	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", url.Values{})

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 2, len(resp))
		assert.Equal(t, "/bin/sh -c bash", resp[0].Command)
		assert.Equal(t, "echo 1", resp[1].Command)
	}
}

func TestProcessesListWithDetached(t *testing.T) {
	// set current provider
	testProvider := &provider.TestProviderRunner{
		Instances: []structs.Instance{
			structs.Instance{},
			structs.Instance{},
			structs.Instance{},
		},
	}
	provider.CurrentProvider = testProvider

	defer func() {
		//TODO: remove: as we arent updating all tests we need tos et current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	// setup expectations on current provider
	testProvider.On("InstanceList").Return(testProvider.Instances, nil)

	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")

	aws := test.StubAws(
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackCycle("convox-test-myapp-staging"),
		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),

		test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
		test.DescribeTasksCycle("convox-test-cluster"),
		test.ListTasksOneoffCycle("convox-test-cluster"),
		test.DescribeTasksOneoffCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),
		test.DescribeTaskDefinitionCycle("convox-test-cluster"),

		test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
		test.DescribeServicesCycle("convox-test-cluster"),
	)
	defer aws.Close()

	docker := test.StubDocker(
		// query for every ECS task to get docker id, command, created
		test.ListECSContainersCycle(),
		test.ListECSOneoffContainersCycle(),

		// query every instance for one-off containers
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersEmptyCycle(),
		test.ListOneoffContainersEmptyCycle(),
	)
	defer docker.Close()

	body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", url.Values{})

	var resp client.Processes
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 2, len(resp))
		assert.Equal(t, "echo 1", resp[0].Command)
		assert.Equal(t, "/bin/sh -c yes", resp[1].Command)
	}
}

// func TestProcessShow(t *testing.T) {}

// func TestProcessStop(t *testing.T) {}

// func TestProcessRun(t *testing.T) {}

// func TestGetProcessesEmpty(t *testing.T) {}

// func TestGetProcessesFailure(t *testing.T) {}
