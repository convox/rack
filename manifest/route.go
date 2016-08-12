package manifest

import "strings"

// Route defines a route on the Manifest
type Route struct {
	ListenerPort int
	Paths        []string
	Port         int
	ServiceName  string
}

// Routes is a list of Routes
type Routes []Route

// RouteListener defines the listener for a Route
type RouteListener struct {
	Port int
}

// Routes returns all routes in the Manifest
func (m Manifest) Routes() Routes {
	routes := Routes{}

	for _, s := range m.Services {
		if path, ok := s.Labels["convox.router.path"]; ok {
			for _, port := range s.Ports {
				routes = append(routes, Route{
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

func (rr Routes) Listeners() []RouteListener {
	ports := map[int]bool{}

	for _, r := range rr {
		ports[r.ListenerPort] = true
	}

	listeners := []RouteListener{}

	for port := range ports {
		listeners = append(listeners, RouteListener{
			Port: port,
		})
	}

	return listeners
}
