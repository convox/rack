package controllers_test

import (
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

func TestProcessesList(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider = &provider.TestProvider{
			Instances: []structs.Instance{
				structs.Instance{},
				structs.Instance{},
				structs.Instance{},
			},
		}

		processes := structs.Processes{
			structs.Process{
				ID:       "foo",
				App:      "myapp-staging",
				Name:     "procname",
				Release:  "R123",
				Command:  "ls -la",
				Host:     "127.0.0.1",
				Image:    "image:tag",
				Instance: "i-1234",
				Ports:    []string{"80", "443"},
				CPU:      0.345,
				Memory:   0.456,
				Started:  time.Unix(1473483567, 0).UTC(),
			},
		}

		// setup expectations on current provider
		models.TestProvider.On("ProcessList", "myapp-staging").Return(processes, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"app\":\"myapp-staging\",\"command\":\"ls -la\",\"cpu\":0.345,\"host\":\"127.0.0.1\",\"id\":\"foo\",\"image\":\"image:tag\",\"instance\":\"i-1234\",\"memory\":0.456,\"name\":\"procname\",\"ports\":[\"80\",\"443\"],\"release\":\"R123\",\"started\":\"2016-09-10T04:59:27Z\"}]")
		}
	})
}

// func TestGetProcessesWithDeployments(t *testing.T) {
//   models.TestProvider = &provider.TestProvider{
//     Instances: []structs.Instance{
//       structs.Instance{},
//       structs.Instance{},
//       structs.Instance{},
//     },
//   }

//   // setup expectations on current provider
//   models.TestProvider.On("InstanceList").Return(models.TestProvider.Instances, nil)

//   aws := test.StubAws(
//     test.DescribeAppStackCycle("convox-test-myapp-staging"),
//     test.DescribeAppStackCycle("convox-test-myapp-staging"),
//     test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

//     test.ListContainerInstancesCycle("convox-test-cluster"),
//     test.DescribeContainerInstancesCycle("convox-test-cluster"),
//     test.DescribeInstancesCycle(),

//     test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
//     test.DescribeTasksCycle("convox-test-cluster"),
//     test.ListTasksOneoffEmptyCycle("convox-test-cluster"),
//     test.DescribeTaskDefinitionCycle("convox-test-cluster"),

//     test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
//     test.DescribeServicesWithDeploymentsCycle("convox-test-cluster"),
//     test.DescribeTaskDefinition3Cycle("convox-test-cluster"),
//     test.DescribeTaskDefinition1Cycle("convox-test-cluster"),
//   )
//   defer aws.Close()

//   docker := test.StubDocker(
//     // query for every ECS task to get docker id, command, created
//     test.ListECSContainersCycle(),

//     // query every instance for one-off containers
//     test.ListOneoffContainersEmptyCycle(),
//     test.ListOneoffContainersEmptyCycle(),
//     test.ListOneoffContainersEmptyCycle(),

//     // query for every container to get CPU and Memory stats
//     test.StatsCycle(),
//   )
//   defer docker.Close()

//   v := url.Values{}
//   v.Add("stats", "true")
//   body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", v)

//   var resp client.Processes
//   err := json.Unmarshal([]byte(body), &resp)

//   if assert.Nil(t, err) {
//     assert.Equal(t, 2, len(resp))
//     assert.Equal(t, "8dfafdbc3a40", resp[0].Id)
//     assert.Equal(t, 0.0974, resp[0].Memory)
//     assert.Equal(t, "pending", resp[1].Id)
//     assert.EqualValues(t, 0, resp[1].Memory)
//   }
// }

// func TestProcessListAttached(t *testing.T) {
//   models.Test(t, func() {
//     models.TestProvider = &provider.TestProvider{
//       Instances: []structs.Instance{
//         structs.Instance{},
//         structs.Instance{},
//         structs.Instance{},
//       },
//     }

//     processes := structs.Processes{
//       structs.Process{
//         ID:       "foo",
//         App:      "myapp-staging",
//         Name:     "procname",
//         Release:  "R123",
//         Command:  "ls -la",
//         Host:     "127.0.0.1",
//         Image:    "image:tag",
//         Instance: "i-1234",
//         Ports:    []string{"80", "443"},
//         Cpu:      0.345,
//         Memory:   0.456,
//         Started:  time.Unix(1473483567, 0).UTC(),
//       },
//     }

//     // setup expectations on current provider
//     models.TestProvider.On("ProcessList", "myapp-staging").Return(processes, nil)

//     hf := test.NewHandlerFunc(controllers.HandlerFunc)

//     if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes", nil)) {
//       hf.AssertCode(t, 200)
//       hf.AssertJSON(t, "[{\"app\":\"myapp-staging\",\"command\":\"ls -la\",\"cpu\":0.345,\"host\":\"127.0.0.1\",\"id\":\"foo\",\"image\":\"image:tag\",\"instance\":\"i-1234\",\"memory\":0.456,\"name\":\"procname\",\"ports\":[\"80\",\"443\"],\"release\":\"R123\",\"started\":\"2016-09-10T04:59:27Z\"}]")
//     }
//   })
// }

// func TestProcessesListWithAttached(t *testing.T) {
//   models.TestProvider = &provider.TestProvider{
//     Instances: []structs.Instance{
//       structs.Instance{},
//       structs.Instance{},
//       structs.Instance{},
//     },
//   }

//   // setup expectations on current provider
//   models.TestProvider.On("InstanceList").Return(models.TestProvider.Instances, nil)

//   aws := test.StubAws(
//     test.DescribeAppStackCycle("convox-test-myapp-staging"),
//     test.DescribeAppStackCycle("convox-test-myapp-staging"),
//     test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

//     test.ListContainerInstancesCycle("convox-test-cluster"),
//     test.DescribeContainerInstancesCycle("convox-test-cluster"),
//     test.DescribeInstancesCycle(),

//     test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
//     test.DescribeTasksCycle("convox-test-cluster"),
//     test.ListTasksOneoffEmptyCycle("convox-test-cluster"),
//     test.DescribeTaskDefinitionCycle("convox-test-cluster"),

//     test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
//     test.DescribeServicesCycle("convox-test-cluster"),
//   )
//   defer aws.Close()

//   docker := test.StubDocker(
//     // query for every ECS task to get docker id, command, created
//     test.ListECSContainersCycle(),

//     // query every instance for one-off containers
//     test.ListOneoffContainersCycle("79bd711b1756"),
//     test.InspectCycle("79bd711b1756"),
//     test.ListOneoffContainersEmptyCycle(),
//   )
//   defer docker.Close()

//   body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", url.Values{})

//   var resp client.Processes
//   err := json.Unmarshal([]byte(body), &resp)

//   if assert.Nil(t, err) {
//     assert.Equal(t, 2, len(resp))
//     assert.Equal(t, "/bin/sh -c bash", resp[0].Command)
//     assert.Equal(t, "echo 1", resp[1].Command)
//   }
// }

// func TestProcessesListWithDetached(t *testing.T) {
//   models.TestProvider = &provider.TestProvider{
//     Instances: []structs.Instance{
//       structs.Instance{},
//       structs.Instance{},
//       structs.Instance{},
//     },
//   }

//   // setup expectations on current provider
//   models.TestProvider.On("InstanceList").Return(models.TestProvider.Instances, nil)

//   os.Setenv("RACK", "convox-test")
//   os.Setenv("CLUSTER", "convox-test-cluster")

//   aws := test.StubAws(
//     test.DescribeAppStackCycle("convox-test-myapp-staging"),
//     test.DescribeAppStackCycle("convox-test-myapp-staging"),
//     test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),

//     test.ListContainerInstancesCycle("convox-test-cluster"),
//     test.DescribeContainerInstancesCycle("convox-test-cluster"),
//     test.DescribeInstancesCycle(),

//     test.ListTasksCycle("convox-test-cluster", "convox-test-myapp-staging-worker-SCELGCIYSKF"),
//     test.DescribeTasksCycle("convox-test-cluster"),
//     test.ListTasksOneoffCycle("convox-test-cluster"),
//     test.DescribeTasksOneoffCycle("convox-test-cluster"),
//     test.DescribeTaskDefinitionCycle("convox-test-cluster"),
//     test.DescribeTaskDefinitionCycle("convox-test-cluster"),

//     test.DescribeAppStackResourcesCycle("convox-test-myapp-staging"),
//     test.DescribeServicesCycle("convox-test-cluster"),
//   )
//   defer aws.Close()

//   docker := test.StubDocker(
//     // query for every ECS task to get docker id, command, created
//     test.ListECSContainersCycle(),
//     test.ListECSOneoffContainersCycle(),

//     // query every instance for one-off containers
//     test.ListOneoffContainersEmptyCycle(),
//     test.ListOneoffContainersEmptyCycle(),
//     test.ListOneoffContainersEmptyCycle(),
//   )
//   defer docker.Close()

//   body := test.HTTPBody("GET", "http://convox/apps/myapp-staging/processes", url.Values{})

//   var resp client.Processes
//   err := json.Unmarshal([]byte(body), &resp)

//   if assert.Nil(t, err) {
//     assert.Equal(t, 2, len(resp))
//     assert.Equal(t, "echo 1", resp[0].Command)
//     assert.Equal(t, "/bin/sh -c yes", resp[1].Command)
//   }
// }

// func TestProcessShow(t *testing.T) {}

// func TestProcessStop(t *testing.T) {}

// func TestProcessRun(t *testing.T) {}

// func TestGetProcessesEmpty(t *testing.T) {}

// func TestGetProcessesFailure(t *testing.T) {}
