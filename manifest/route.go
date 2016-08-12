package manifest

import "strings"

type ManifestRoute struct {
	ListenerPort int
	Paths        []string
	Port         int
	ServiceName  string
}

type ManifestRoutes []ManifestRoute

type ManifestRouteListener struct {
	Port int
}

func (m Manifest) Routes() ManifestRoutes {
	routes := ManifestRoutes{}

	for _, s := range m.Services {
		if path, ok := s.Labels["convox.router.path"]; ok {
			for _, port := range s.Ports {
				routes = append(routes, ManifestRoute{
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

func (rr ManifestRoutes) Listeners() []ManifestRouteListener {
	ports := map[int]bool{}

	for _, r := range rr {
		ports[r.ListenerPort] = true
	}

	listeners := []ManifestRouteListener{}

	for port := range ports {
		listeners = append(listeners, ManifestRouteListener{
			Port: port,
		})
	}

	return listeners
}
