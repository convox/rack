package manifest

import (
	"fmt"
	"sort"
)

const ServiceMapSizeIncrement = 5

// Group - A service group
type Group struct {
	Name           string
	Services       []*Service
	ServiceMap     map[string]*Service
	serviceMapSize uint16
}

// NewGroup - Creates a new group
func NewGroup(name string) Group {
	serviceMap := make(map[string]*Service, ServiceMapSizeIncrement)
	var services []*Service
	return Group{
		Name:       name,
		ServiceMap: serviceMap,
		Services:   services,
	}
}

// AddService - Adds a service to a group
func (g *Group) AddService(service Service) {
	g.Services = append(g.Services, &service)

	// Add the service to the service map.
	// If it's too big then grow the map
	if len(g.Services)%ServiceMapSizeIncrement == 0 {
		newServiceMap := make(map[string]*Service, g.serviceMapSize+ServiceMapSizeIncrement)
		for serviceName, service := range g.ServiceMap {
			newServiceMap[serviceName] = service
		}
		g.ServiceMap = newServiceMap
	}
	g.ServiceMap[service.Name] = &service
}

// ParamName - Returns the Param name to be used for the group's var
func (g Group) ParamName(name string) string {
	return fmt.Sprintf("%s%sGroup", UpperName(g.Name), name)
}

// ServicesWithLoadBalancers - Returns a slice of services that have load balancers
func (g Group) ServicesWithLoadBalancers() []*Service {
	var services []*Service
	for _, service := range g.Services {
		if service.HasBalancer() {
			services = append(services, service)
		}
	}
	return services
}

// Links - Lists the links for a group
func (g Group) Links(serviceName string) []string {
	matchingService, matchingServiceExists := g.ServiceMap[serviceName]
	if !matchingServiceExists {
		return nil
	}
	links := matchingService.Links
	var filteredLinks []string
	for _, link := range links {
		if _, ok := g.ServiceMap[link]; ok {
			filteredLinks = append(filteredLinks, link)
		}
	}
	return filteredLinks
}

// HasLink - Returns true if a service has a link to another service in the group
func (g *Group) HasLink(serviceName string, searchLink string) bool {
	for _, link := range g.Links(serviceName) {
		if link == searchLink {
			return true
		}
	}
	return false
}

// HasService - Returns true if the service exists in this group
func (g *Group) HasService(serviceName string) bool {
	_, ok := g.ServiceMap[serviceName]
	return ok
}

// HasBalancer - Returns true if this group has any load balancers associated with it
func (g Group) HasBalancer() bool {
	for _, service := range g.Services {
		if service.HasBalancer() {
			return true
		}
	}
	return false
}

// DeploymentMinimum - Returns the deployment minimum of the current group
func (g Group) DeploymentMinimum() string {
	return "100"
}

// DeploymentMaximum - Returns the deployment maximum of the current group
func (g Group) DeploymentMaximum() string {
	return "200"
}

func (m *Manifest) sortedServiceNames() []string {
	var serviceNames []string

	for serviceName := range m.Services {
		serviceNames = append(serviceNames, serviceName)
	}

	sort.Strings(serviceNames)
	return serviceNames
}

// ServiceGroups - Returns all the groups defined in a manifest
func (m Manifest) ServiceGroups() []*Group {
	var groups []*Group

	groupMap := make(map[string]*Group, len(m.Services))

	for _, serviceName := range m.sortedServiceNames() {
		service := m.Services[serviceName]
		if groupName, ok := service.Labels["convox.group"]; ok {
			groups = addOrUpdateGroup(groupName, service, groupMap, groups)
		} else {
			groups = addOrUpdateGroup(serviceName, service, groupMap, groups)
		}
	}

	return groups
}

// ServiceGroup - Returns a specific group
func (m Manifest) ServiceGroup(name string) *Group {
	group := NewGroup(name)
	for _, serviceName := range m.sortedServiceNames() {
		service := m.Services[serviceName]

		if service.GroupName() == name {
			group.AddService(service)
		}
	}
	return &group
}

// GetGroupForServiceName - Retrieves the group for a specific service name
func (m *Manifest) GetGroupForServiceName(serviceName string) (*Group, error) {
	service, ok := m.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("Service `%s` does not exist", serviceName)
	}

	if groupName, ok := service.Labels["convox.group"]; ok {
		return m.ServiceGroup(groupName), nil
	}
	return m.ServiceGroup(serviceName), nil
}

func addOrUpdateGroup(groupName string, service Service, groupMap map[string]*Group, groups []*Group) []*Group {
	if group, ok := groupMap[groupName]; ok {
		group.AddService(service)
	} else {
		group := NewGroup(groupName)
		group.AddService(service)
		groups = append(groups, &group)
		groupMap[groupName] = &group
	}
	return groups
}
