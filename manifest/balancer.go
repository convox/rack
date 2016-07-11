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

type ManifestPort struct {
	Balancer  string
	Container string
	Public    bool
}

type ManifestBalancer struct {
	Entry  Service
	Public bool
}

func (m Manifest) Balancers() []ManifestBalancer {
	balancers := []ManifestBalancer{}

	for _, entry := range m.Services {
		if len(entry.Ports) > 0 {
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

func (m Manifest) HasProcesses() bool {
	return len(m.Services) > 0
}

func (mb ManifestBalancer) ExternalPorts() []Port {
	return mb.Entry.ExternalPorts()
}

func (mb ManifestBalancer) FirstPort() string {
	if ports := mb.PortMappings(); len(ports) > 0 {
		return strconv.Itoa(ports[0].Balancer)
	}

	return ""
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

	// Unbound apps use legacy StackName or StackName-ProcessName format
	//TODO
	// if mb.Entry.primary {
	// 	return template.HTML(`{ "Ref": "AWS::StackName" }`)
	// }

	if mb.Public {
		return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s" ] ] }`, mb.ProcessName()))
	}

	return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s", "i" ] ] }`, mb.ProcessName()))
}

func (mb ManifestBalancer) InternalPorts() []Port {
	fmt.Printf("mb.Entry.InternalPorts(): %+v\n", mb.Entry.InternalPorts())
	return mb.Entry.InternalPorts()
}

func (mb ManifestBalancer) Ports() []string {
	pp := mb.Entry.Ports
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
	return mb.Entry.Ports
}

func (mb ManifestBalancer) Scheme() string {
	if mb.Public {
		return "internet-facing"
	}

	return "internal"
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
