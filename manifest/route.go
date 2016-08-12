package manifest

import "strings"

type manifestRoute struct {
	ListenerPort int
	Paths        []string
	Port         int
	ServiceName  string
}

type manifestRoutes []manifestRoute

type manifestRouteListener struct {
	Port int
}

// Routes returns all routes in the Manifest
func (m Manifest) Routes() manifestRoutes {
	routes := manifestRoutes{}

	for _, s := range m.Services {
		if path, ok := s.Labels["convox.router.path"]; ok {
			for _, port := range s.Ports {
				routes = append(routes, manifestRoute{
					ListenerPort: port.Balancer,
					Port:         port.Container,
					Paths:        strings.Split(path, ","),
					ServiceName:  s.Name,
				})
			}
		}
	}

	return routes
}

func (rr manifestRoutes) Listeners() []manifestRouteListener {
	ports := map[int]bool{}

	for _, r := range rr {
		ports[r.ListenerPort] = true
	}

	listeners := []manifestRouteListener{}

	for port := range ports {
		listeners = append(listeners, manifestRouteListener{
			Port: port,
		})
	}

	return listeners
}
