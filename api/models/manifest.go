package models

import (
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

// set to false when testing for deterministic ports
var ManifestRandomPorts = true

type Manifest []ManifestEntry

type ManifestEntry struct {
	Name string

	Build     string                   `yaml:"build"`
	Command   interface{}              `yaml:"command"`
	Env       []string                 `yaml:"environment"`
	Exports   map[string]string        `yaml:"-"`
	Image     string                   `yaml:"image"`
	Links     []string                 `yaml:"links"`
	LinkVars  map[string]template.HTML `yaml:"-"`
	*Manifest `yaml:"-"`
	Ports     []string `yaml:"ports"`
	Volumes   []string `yaml:"volumes"`

	primary bool
	randoms map[string]int
}

type ManifestPort struct {
	Balancer  string
	Container string
	Public    bool
}

type ManifestEntries map[string]ManifestEntry

type ManifestBalancer struct {
	Entry  ManifestEntry
	Public bool
}

func LoadManifest(data string) (Manifest, error) {
	var entries ManifestEntries

	err := yaml.Unmarshal([]byte(data), &entries)

	if err != nil {
		return nil, fmt.Errorf("invalid manifest: %s", err)
	}

	names := []string{}

	for name, _ := range entries {
		names = append(names, name)
	}

	sort.Strings(names)

	manifest := make(Manifest, 0)

	currentPort := 5000

	for _, name := range names {
		entry := entries[name]
		entry.Manifest = &manifest
		entry.Name = name
		entry.randoms = make(map[string]int)

		for _, port := range entry.Ports {
			p := strings.Split(port, ":")[0]

			if ManifestRandomPorts {
				entry.randoms[p] = rand.Intn(62000) + 3000
			} else {
				entry.randoms[p] = currentPort
				currentPort += 1
			}
		}

		manifest = append(manifest, ManifestEntry(entry))
	}

	return manifest, nil
}

func (m Manifest) BalancerResourceName(process string) string {
	for _, b := range m.Balancers() {
		if b.Entry.Name == process {
			return b.ResourceName()
		}
	}

	return ""
}

func (m Manifest) Entry(name string) *ManifestEntry {
	for _, me := range m {
		if me.Name == name {
			return &me
		}
	}

	return nil
}

func (m Manifest) EntryNames() []string {
	names := make([]string, len(m))

	for i, entry := range m {
		names[i] = entry.Name
	}

	return names
}

func (m Manifest) Balancers() []ManifestBalancer {
	balancers := []ManifestBalancer{}

	for _, entry := range m {
		if len(entry.PortMappings()) > 0 {
			balancers = append(balancers, ManifestBalancer{
				Entry:  entry,
				Public: len(entry.InternalPorts()) == 0,
			})
		}
	}

	return balancers
}

func (m Manifest) Formation() (string, error) {
	data, err := buildTemplate("app", "app", m)

	if err != nil {
		return "", err
	}

	pretty, err := prettyJson(string(data))

	if err != nil {
		return "", err
	}

	return pretty, nil
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
	if len(m) == 0 {
		return true // special case to pre-initialize ELB at app create
	}

	for _, me := range m {
		if len(me.ExternalPorts()) > 0 {
			return true
		}
	}

	return false
}

func (m Manifest) HasProcesses() bool {
	return len(m) > 0
}

func (mb ManifestBalancer) ExternalPorts() []string {
	ep := mb.Entry.ExternalPorts()
	sp := make([]string, len(ep))

	for i, p := range ep {
		sp[i] = strings.Split(p, ":")[0]
	}

	return sp
}

func (mb ManifestBalancer) FirstPort() string {
	if ports := mb.PortMappings(); len(ports) > 0 {
		return ports[0].Balancer
	}

	return ""
}

func (mb ManifestBalancer) LoadBalancerName() template.HTML {
	if mb.Entry.primary {
		return template.HTML(`{ "Ref": "AWS::StackName" }`)
	}

	if mb.Public {
		return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s" ] ] }`, mb.ProcessName()))
	}

	return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s", "internal" ] ] }`, mb.ProcessName()))
}

func (mb ManifestBalancer) InternalPorts() []string {
	fmt.Printf("mb.Entry.InternalPorts(): %+v\n", mb.Entry.InternalPorts())
	return mb.Entry.InternalPorts()
}

func (mb ManifestBalancer) Ports() []string {
	pp := mb.Entry.Ports
	sp := make([]string, len(pp))

	for i, p := range pp {
		sp[i] = strings.Split(p, ":")[0]
	}

	return sp
}

func (mb ManifestBalancer) ProcessName() string {
	return mb.Entry.Name
}

func (mb ManifestBalancer) Randoms() map[string]int {
	return mb.Entry.Randoms()
}

func (mb ManifestBalancer) ResourceName() string {
	if mb.Entry.primary {
		return "Balancer"
	}

	var suffix string
	if !mb.Public {
		suffix = "Internal"
	}

	return "Balancer" + UpperName(mb.Entry.Name) + suffix
}

func (mb ManifestBalancer) PortMappings() []ManifestPort {
	return mb.Entry.PortMappings()
}

func (mb ManifestBalancer) Scheme() string {
	if mb.Public {
		return "internet-facing"
	}

	return "internal"
}

func (me *ManifestEntry) BalancerResourceName() string {
	return ""
}

func (me *ManifestEntry) CommandString() string {
	switch cmd := me.Command.(type) {
	case nil:
		return ""
	case string:
		return cmd
	case []interface{}:
		parts := make([]string, len(cmd))

		for i, c := range cmd {
			parts[i] = c.(string)
		}

		return strings.Join(parts, " ")
	default:
		fmt.Fprintf(os.Stderr, "unexpected type for command: %T\n", cmd)
		return ""
	}
}

func (me ManifestEntry) InternalPorts() []string {
	internal := []string{}

	for _, port := range me.Ports {
		if len(strings.Split(port, ":")) == 1 {
			internal = append(internal, port)
		}
	}

	return internal
}

func (me ManifestEntry) ContainerPorts() []string {
	extmap := map[string]bool{}
	ext := []string{}

	for _, port := range me.Ports {
		if parts := strings.Split(port, ":"); len(parts) == 2 {
			extmap[parts[1]] = true
		}
	}

	for k, _ := range extmap {
		ext = append(ext, k)
	}

	sort.Strings(ext)

	return ext
}

func (me ManifestEntry) EnvMap() map[string]string {
	envs := map[string]string{}

	for _, env := range me.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envs[parts[0]] = parts[1]
		}
	}

	return envs
}

func (me ManifestEntry) MountableVolumes() []string {
	volumes := []string{}

	for _, volume := range me.Volumes {
		if strings.HasPrefix(volume, "/var/run/docker.sock") {
			volumes = append(volumes, volume)
		}
	}

	return volumes
}

func (me ManifestEntry) HasBalancer() bool {
	return len(me.PortMappings()) > 0
}

func (me ManifestEntry) PortMappings() []ManifestPort {
	mappings := []ManifestPort{}

	for _, port := range me.Ports {
		parts := strings.SplitN(port, ":", 2)

		switch len(parts) {
		case 1:
			mappings = append(mappings, ManifestPort{
				Balancer:  parts[0],
				Container: parts[0],
				Public:    false,
			})
		case 2:
			mappings = append(mappings, ManifestPort{
				Balancer:  parts[0],
				Container: parts[1],
				Public:    true,
			})
		}
	}

	return mappings
}

func (me ManifestEntry) ExternalPorts() []string {
	ext := []string{}

	for _, port := range me.Ports {
		if len(strings.Split(port, ":")) == 2 {
			ext = append(ext, port)
		}
	}

	return ext
}

func (me ManifestEntry) Randoms() map[string]int {
	return me.randoms
}
