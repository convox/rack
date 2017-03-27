package manifest_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

func assertExpectedServiceNames(t *testing.T, expectedNames []string, services []*manifest.Service) {
	for i, expectedName := range expectedNames {
		serviceName := services[i].Name
		assert.Equal(t, expectedName, serviceName, fmt.Sprintf("should have expected service name in order"))
	}
}

func TestListGroups(t *testing.T) {
	m, err := manifestFixture("group")

	if assert.NoError(t, err) {
		groups := m.ServiceGroups()
		if assert.Equal(t, len(groups), 2) {
			webGroup := groups[0]
			workerGroup := groups[1]

			// Get groups again to ensure that service groups returns consistently
			groupsAgain := m.ServiceGroups()

			assert.Equal(t, 2, len(groupsAgain), "manifest.ServiceGroups should return consistent results")
			assert.EqualValues(t, groups, groupsAgain, "manifest.ServiceGroups should return consistent results")
			assert.Equal(t, groups[0].Name, groupsAgain[0].Name, "manifest.ServiceGroups should return consistent results")
			assert.Equal(t, groups[1].Name, groupsAgain[1].Name, "manifest.ServiceGroups should return consistent results")
			assert.Equal(t, groups[0].Services, groupsAgain[0].Services, "manifest.ServiceGroups should return consistent results")
			assert.Equal(t, groups[1].Services, groupsAgain[1].Services, "manifest.ServiceGroups should return consistent results")

			assert.Equal(t, "web", webGroup.Name, `The first group returned by ServiceGroups should be "web"`)
			assert.Equal(t, "worker", workerGroup.Name, `The second group returned by ServiceGroups should be "worker"`)

			assertExpectedServiceNames(t, []string{"reverse-proxy", "web"}, webGroup.Services)
			assertExpectedServiceNames(t, []string{"worker"}, workerGroup.Services)
		}
	}
}

func TestSingleGroupRetrieval(t *testing.T) {
	m, err := manifestFixture("group")

	if assert.NoError(t, err) {
		group := m.ServiceGroup("web")

		assert.Equal(t, 2, len(group.Services), "Web group should have two services in it")

		assertExpectedServiceNames(t, []string{"reverse-proxy", "web"}, group.Services)

		// Check ParamName method
		assert.Equal(t, "WebFooGroup", group.ParamName("Foo"), "should generate expected param name")
		assertExpectedServiceNames(t, []string{"reverse-proxy"}, group.ServicesWithLoadBalancers())

		// Check Links method
		assert.EqualValues(t, []string{"web"}, group.Links("reverse-proxy"), "reverse proxy should have expected links")
	}
}

func TestGetGroupForServiceName(t *testing.T) {
	m, err := manifestFixture("group")

	if assert.NoError(t, err) {
		group, err := m.GetGroupForServiceName("web")

		if assert.NoError(t, err) {
			assert.Equal(t, 2, len(group.Services), "Web group should have two services in it")
			assertExpectedServiceNames(t, []string{"reverse-proxy", "web"}, group.Services)
		}
	}
}

func makeFakeService(name string, links []string) manifest.Service {
	return manifest.Service{
		Name:  name,
		Links: links,
	}
}

func makeFakeServiceWithLoadBalancer(name string, links []string, port int) manifest.Service {
	service := makeFakeService(name, links)
	service.Ports = manifest.Ports{
		manifest.Port{
			Name:      "port",
			Balancer:  port,
			Container: port,
			Protocol:  "tcp",
			Public:    true,
		},
	}
	return service
}

func setupGroupForTest() manifest.Group {
	group := manifest.NewGroup("group")

	// Notice "service4"" doesn't exist in this group. That's intentional for the Group#Links() test below
	group.AddService(makeFakeService("service1", []string{"service2", "service3", "service4"}))
	group.AddService(makeFakeService("service2", []string{"service3"}))
	group.AddService(makeFakeService("service3", []string{"service4"}))
	return group
}

func setupGroupForTestWithLoadBalancers() manifest.Group {
	group := setupGroupForTest()

	service4 := makeFakeServiceWithLoadBalancer("service4", nil, 9000)
	service5 := makeFakeServiceWithLoadBalancer("service5", nil, 9001)

	group.AddService(service4)
	group.AddService(service5)

	return group
}

func TestGroupAddServiceMethod(t *testing.T) {
	group := manifest.NewGroup("group")

	group.AddService(makeFakeService("service1", nil))
	group.AddService(makeFakeService("service2", nil))
	group.AddService(makeFakeService("service3", nil))

	assert.Equal(t, 3, len(group.Services), "group should have expected size")
}

func TestGroupLinksMethod(t *testing.T) {
	group := setupGroupForTest()

	assert.Equal(t, []string{"service2", "service3"}, group.Links("service1"), "service1 should have expected links")
	assert.Equal(t, []string{"service3"}, group.Links("service2"), "service2 should have expected links")
	assert.Equal(t, 0, len(group.Links("service3")), "service3 should have expected links")
}

func TestGroupHasLinksMethod(t *testing.T) {
	group := setupGroupForTest()

	assert.True(t, group.HasLink("service2", "service3"))
	assert.False(t, group.HasLink("service2", "service4"))
}

func TestGroupHasServiceMethod(t *testing.T) {
	group := setupGroupForTest()

	assert.True(t, group.HasService("service1"))
	assert.False(t, group.HasService("service200"))
}

func TestGroupHasBalancer(t *testing.T) {
	groupWithoutLoadBalancer := setupGroupForTest()
	groupWithLoadBalancer := setupGroupForTestWithLoadBalancers()

	assert.False(t, groupWithoutLoadBalancer.HasBalancer())
	assert.True(t, groupWithLoadBalancer.HasBalancer())
}

func TestGroupServicesWithLoadBalancers(t *testing.T) {
	group := setupGroupForTestWithLoadBalancers()

	servicesWithLoadBalancers := group.ServicesWithLoadBalancers()
	assert.Equal(t, 2, len(servicesWithLoadBalancers))
	assertExpectedServiceNames(t, []string{"service4", "service5"}, servicesWithLoadBalancers)
}
