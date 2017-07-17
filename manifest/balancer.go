package manifest

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
	// "github.com/convox/rack/api/models"
)

type ManifestBalancer struct {
	Entry  Service
	Public bool
}

func (m Manifest) Balancers() []ManifestBalancer {
	balancers := []ManifestBalancer{}

	for _, entry := range m.Services {
		if entry.HasBalancer() {
			balancers = append(balancers, ManifestBalancer{
				Entry:  entry,
				Public: len(entry.InternalPorts()) == 0,
			})
		}
	}

	return balancers
}

func (m Manifest) GetBalancer(name string) *ManifestBalancer {
	for _, mb := range m.Balancers() {
		if mb.Entry.Name == name {
			return &mb
		}
	}

	return nil
}

func (m Manifest) HasProcesses() bool {
	return len(m.Services) > 0
}

func (mb ManifestBalancer) LoadBalancerName(bound bool, appName string) template.HTML {
	// Bound apps do not use the StackName directly and ignore Entry.primary
	// and use AppName-EntryName-RackAppEntryHash format
	if bound {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", os.Getenv("RACK"), appName, mb.Entry.Name)))
		prefix := fmt.Sprintf("%s-%s", appName, mb.Entry.Name)
		suffix := "-" + base32.StdEncoding.EncodeToString(hash[:])[:7]
		if !mb.Public {
			suffix += "-i"
		}
		// ELB name must be 32 chars or less
		if len(prefix) > 32-len(suffix) {
			prefix = prefix[:32-len(suffix)]
		}
		return template.HTML(`"` + prefix + suffix + `"`)
	}

	//Unbound apps use legacy StackName or StackName-ProcessName format
	//TODO I'm not sure this will be set at the time of calling
	if mb.Entry.Primary {
		return template.HTML(`{ "Ref": "AWS::StackName" }`)
	}

	if mb.Public {
		return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s" ] ] }`, mb.ProcessName()))
	}

	return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s", "i" ] ] }`, mb.ProcessName()))
}

// HasExternalPorts returns true if the Manifest's Services have external ports
func (m Manifest) HasExternalPorts() bool {
	if len(m.Services) == 0 {
		return true // special case to pre-initialize ELB at app create
	}

	for _, me := range m.Services {
		if len(me.ExternalPorts()) > 0 {
			return true
		}
	}

	return false
}

// InternalPorts returns a collection of Port structs of the Manifest's internal ports
func (mb ManifestBalancer) InternalPorts() []Port {
	return mb.Entry.InternalPorts()
}

// ExternalPorts returns a collection of Port structs of the Manifest's external ports
func (mb ManifestBalancer) ExternalPorts() []Port {
	return mb.Entry.ExternalPorts()
}

// FirstPort returns the first TCP Port defined on the first Service in the Manifest
func (mb ManifestBalancer) FirstPort() string {
	if ports := mb.PortMappings(); len(ports) > 0 {
		return strconv.Itoa(ports[0].Balancer)
	}

	return ""
}

func (mb ManifestBalancer) Ports() []string {
	pp := mb.Entry.TCPPorts()
	sp := make([]string, len(pp))

	for _, p := range pp {
		sp = append(sp, strconv.Itoa(p.Balancer))
	}

	return sp
}

func (mb ManifestBalancer) ProcessName() string {
	return mb.Entry.Name
}

func (mb ManifestBalancer) ResourceName() string {
	// unbound apps special case the balancer name for the primary proces
	//TODO work out what this means
	// if mb.Entry.primary {
	// 	if mb.Entry.app == nil || !mb.Entry.app.IsBound() {
	// 		return "Balancer"
	// 	}
	// }

	var suffix string
	if !mb.Public {
		suffix = "Internal"
	}

	return "Balancer" + UpperName(mb.Entry.Name) + suffix
}

func (mb ManifestBalancer) PortMappings() []Port {
	return mb.Entry.TCPPorts()
}

func (mb ManifestBalancer) Scheme() string {
	if mb.Public {
		return "internet-facing"
	}

	return "internal"
}

// Protocol returns the desired listener protocol of the balancer
func (mb ManifestBalancer) Protocol(p Port) string {
	return mb.Entry.Labels[fmt.Sprintf("convox.port.%d.protocol", p.Balancer)]
}

// ListenerProtocol returns the protocol the balancer should use to listen
func (mb ManifestBalancer) ListenerProtocol(p Port) string {
	switch mb.Protocol(p) {
	case "tls":
		return "SSL"
	case "tcp":
		return "TCP"
	case "https":
		return "HTTPS"
	case "http":
		return "HTTP"
	}
	return "TCP"
}

// InstanceProtocol returns protocol the container is listening with
func (mb ManifestBalancer) InstanceProtocol(p Port) string {
	secure := mb.Entry.Labels[fmt.Sprintf("convox.port.%d.secure", p.Balancer)] == "true"

	switch mb.Protocol(p) {
	case "tcp", "tls":
		if secure {
			return "SSL"
		}
		return "TCP"
	case "https", "http":
		if secure {
			return "HTTPS"
		}
		return "HTTP"
	}

	return "TCP"
}

// ProxyProtocol returns true if the container is listening for PROXY protocol
func (mb ManifestBalancer) ProxyProtocol(p Port) bool {
	return mb.Entry.Labels[fmt.Sprintf("convox.port.%d.proxy", p.Balancer)] == "true"
}

func UpperName(name string) string {
	// myapp -> Myapp; my-app -> MyApp
	us := strings.ToUpper(name[0:1]) + name[1:]

	for {
		i := strings.Index(us, "-")

		if i == -1 {
			break
		}

		s := us[0:i]

		if len(us) > i+1 {
			s += strings.ToUpper(us[i+1 : i+2])
		}

		if len(us) > i+2 {
			s += us[i+2:]
		}

		us = s
	}

	return us
}

func (mb ManifestBalancer) Randoms() map[string]int {
	return mb.Entry.Randoms()
}

// HealthPath The path to check for health. If unset, then implies TCP check
func (mb ManifestBalancer) HealthPath() string {
	return mb.Entry.Labels["convox.health.path"]
}

// HealthPort The balancer port that maps to the container port specified in
// manifest
func (mb ManifestBalancer) HealthPort() string {
	if len(mb.Entry.TCPPorts()) == 0 {
		return ""
	}

	if port := mb.Entry.Labels["convox.health.port"]; port != "" {
		for _, p := range mb.Entry.TCPPorts() {
			if strconv.Itoa(p.Container) == port {
				return strconv.Itoa(p.Balancer)
			}
		}

		// couldnt find the port they are talking about
		return ""
	}

	return coalesce(mb.Entry.Labels["convox.health.port"], strconv.Itoa(mb.Entry.TCPPorts()[0].Balancer))
}

// HealthProtocol returns the protocol to use for the health check
func (mb ManifestBalancer) HealthProtocol() string {
	secure := mb.Entry.Labels[fmt.Sprintf("convox.port.%s.secure", mb.HealthPort())] == "true"

	if path := mb.Entry.Labels["convox.health.path"]; path != "" {
		if secure {
			return "HTTPS"
		}
		return "HTTP"
	}

	if secure {
		return "SSL"
	}
	return "TCP"
}

// HealthTimeout The default health timeout when one is not specified
func (mb ManifestBalancer) HealthTimeout() string {
	if timeout := mb.Entry.Labels["convox.health.timeout"]; timeout != "" {
		return timeout
	}
	return "3"
}

// HealthInterval The amount of time in between health checks.
// This is derived from the timeout value, which must be less than the interval
func (mb ManifestBalancer) HealthInterval() (string, error) {
	timeout := mb.HealthTimeout()
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		return "", err
	}
	interval := strconv.Itoa(timeoutInt + 2)
	return interval, nil
}

// HealthThresholdHealthy The number of consecutive successful health checks
// that must occur before declaring an EC2 instance healthy.
func (mb ManifestBalancer) HealthThresholdHealthy() string {
	if threshold := mb.Entry.Labels["convox.health.threshold.healthy"]; threshold != "" {
		return threshold
	}
	return "2"
}

// HealthThresholdUnhealthy The number of consecutive failed health checks that
// must occur before declaring an EC2 instance unhealthy
func (mb ManifestBalancer) HealthThresholdUnhealthy() string {
	if threshold := mb.Entry.Labels["convox.health.threshold.unhealthy"]; threshold != "" {
		return threshold
	}
	return "2"
}

// IdleTimeout The amount of time to allow the balancer to keep idle connections open. This should be
// greater than the keep-alive timeout on your back-end, so that the balancer is responsible for
// closing connections
func (mb ManifestBalancer) IdleTimeout() (string, error) {
	if timeout := mb.Entry.Labels["convox.idle.timeout"]; timeout != "" {
		timeoutInt, err := strconv.Atoi(timeout)
		if err != nil {
			return "", err
		}
		if timeoutInt < 1 || timeoutInt > 3600 {
			return "", fmt.Errorf("convox.idle.timeout must be between 1 and 3600")
		}
		return timeout, nil
	}
	return "3600", nil
}

// DrainingTimeout The amount of time to allow a draining balancer to keep active connections open.
func (mb ManifestBalancer) DrainingTimeout() (string, error) {
	if timeout := mb.Entry.Labels["convox.draining.timeout"]; timeout != "" {
		timeoutInt, err := strconv.Atoi(timeout)
		if err != nil {
			return "", err
		}
		if timeoutInt < 1 || timeoutInt > 3600 {
			return "", fmt.Errorf("convox.draining.timeout must be between 1 and 3600")
		}
		return timeout, nil
	}
	return "60", nil
}
