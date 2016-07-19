package models

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// set to false when testing for deterministic ports
var ManifestRandomPorts = true

type Manifest []ManifestEntry

type ManifestEntry struct {
	Name string

	Build      string                   `yaml:"build"`
	Command    interface{}              `yaml:"command"`
	Env        MapOrEqualSlice          `yaml:"environment"`
	Exports    map[string]string        `yaml:"-"`
	Image      string                   `yaml:"image"`
	Labels     interface{}              `yaml:"labels"`
	Links      []string                 `yaml:"links"`
	LinkVars   map[string]template.HTML `yaml:"-"`
	Ports      []string                 `yaml:"ports"`
	Privileged bool                     `yaml:"privileged"`
	Volumes    []string                 `yaml:"volumes"`

	app     *App
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

func LoadManifest(data string, app *App) (Manifest, error) {
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
		entry.Name = name
		// This could be nil
		entry.app = app
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

func (m Manifest) Rack() string {
	return os.Getenv("RACK")
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

func (m Manifest) CronJobs() []CronJob {
	cronjobs := []CronJob{}

	for _, entry := range m {
		labels := entry.LabelsByPrefix("convox.cron")
		for key, value := range labels {
			cronjob := NewCronJobFromLabel(key, value)
			cronjob.ManifestEntry = entry
			cronjobs = append(cronjobs, cronjob)
		}
	}
	return cronjobs
}

func (m Manifest) AppName() string {
	return m[0].app.Name
}

// LabelsByPrefix will return the labels that have a given prefix
func (e ManifestEntry) LabelsByPrefix(prefix string) map[string]string {
	returnLabels := make(map[string]string)
	switch labels := e.Labels.(type) {
	case map[interface{}]interface{}:
		for k, v := range labels {
			ks, ok := k.(string)

			if !ok {
				continue
			}

			vs, ok := v.(string)

			if !ok {
				continue
			}

			if strings.HasPrefix(ks, prefix) {
				returnLabels[ks] = vs
			}
		}
	case []interface{}:
		for _, label := range labels {
			ls, ok := label.(string)

			if !ok {
				continue
			}

			if parts := strings.SplitN(ls, "=", 2); len(parts) == 2 {
				if strings.HasPrefix(parts[0], prefix) {
					returnLabels[parts[0]] = parts[1]
				}
			}
		}
	}
	return returnLabels
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
	// Bound apps do not use the StackName directly and ignore Entry.primary
	// and use AppName-EntryName-RackAppEntryHash format
	if mb.Entry.app != nil && mb.Entry.app.IsBound() {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", os.Getenv("RACK"), mb.Entry.app.Name, mb.Entry.Name)))
		prefix := fmt.Sprintf("%s-%s", mb.Entry.app.Name, mb.Entry.Name)
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
	if mb.Entry.primary {
		return template.HTML(`{ "Ref": "AWS::StackName" }`)
	}

	if mb.Public {
		return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s" ] ] }`, mb.ProcessName()))
	}

	return template.HTML(fmt.Sprintf(`{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "%s", "i" ] ] }`, mb.ProcessName()))
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
	// unbound apps special case the balancer name for the primary proces
	if mb.Entry.primary {
		if mb.Entry.app == nil || !mb.Entry.app.IsBound() {
			return "Balancer"
		}
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
	case string:
		return cmd
	default:
		return ""
	}
}

func (me *ManifestEntry) CommandArray() []string {
	switch cmd := me.Command.(type) {
	case nil:
		return []string{}
	case string:
		return []string{}
	case []interface{}:
		commands := make([]string, len(cmd))
		for i, c := range cmd {
			commands[i] = c.(string)
		}
		return commands
	default:
		fmt.Fprintf(os.Stderr, "unexpected type for command: %T\n", cmd)
		return []string{}
	}
}

func (me ManifestEntry) Label(key string) string {
	switch labels := me.Labels.(type) {
	case map[interface{}]interface{}:
		for k, v := range labels {
			ks, ok := k.(string)

			if !ok {
				return ""
			}

			vs, ok := v.(string)

			if !ok {
				return ""
			}

			if ks == key {
				return vs
			}
		}
	case []interface{}:
		for _, label := range labels {
			ls, ok := label.(string)

			if !ok {
				return ""
			}

			if parts := strings.SplitN(ls, "=", 2); len(parts) == 2 {
				if parts[0] == key {
					return parts[1]
				}
			}
		}
	}

	return ""
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

	// expose app and rack metadata to the container environment
	// user supplied environment, either in the manifest or with `convox env set` will override
	if me.app != nil {
		envs["APP"] = me.app.Name
	}
	envs["RACK"] = os.Getenv("RACK")
	envs["AWS_REGION"] = os.Getenv("AWS_REGION")

	for _, env := range me.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envs[parts[0]] = parts[1]
		}
	}

	return envs
}

// MountableVolume describes a mountable volume
type MountableVolume struct {
	Host      string
	Container string
}

// MountableVolumes return the mountable volumes for a manifest entry
func (e ManifestEntry) MountableVolumes() []MountableVolume {
	volumes := []MountableVolume{}

	for _, volume := range e.Volumes {
		parts := strings.Split(volume, ":")

		// if only one volume part use it for both sides
		if len(parts) == 1 {
			parts = append(parts, parts[0])
		}

		// if we dont have two volume parts bail
		if len(parts) != 2 {
			continue
		}

		// only support absolute paths for volume source
		if !filepath.IsAbs(parts[0]) {
			continue
		}

		volumes = append(volumes, MountableVolume{
			Host:      parts[0],
			Container: parts[1],
		})
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

func (me ManifestEntry) RegistryImage(app *App, buildId string) string {
	if registryId := app.Outputs["RegistryId"]; registryId != "" {
		return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), app.Outputs["RegistryRepository"], me.Name, buildId)
	}

	return fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), app.Name, me.Name, buildId)
}

type MapOrEqualSlice []string

// UnmarshalYAML implements the Unmarshaller interface.
func (s *MapOrEqualSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value interface{}
	err := unmarshal(&value)
	if err != nil {
		return err
	}

	parts, err := unmarshalToStringOrSepMapParts(value, "=")
	if err != nil {
		return err
	}
	*s = parts
	return nil
}

// the following code is from libcompose/yaml/types_yaml (Apache License)
func unmarshalToStringOrSepMapParts(value interface{}, key string) ([]string, error) {
	switch value := value.(type) {
	case []interface{}:
		return toStrings(value)
	case map[interface{}]interface{}:
		return toSepMapParts(value, key)
	default:
		return nil, fmt.Errorf("Failed to unmarshal Map or Slice: %#v", value)
	}
}

// the following code is from libcompose/yaml/types_yaml (Apache License)
func toSepMapParts(value map[interface{}]interface{}, sep string) ([]string, error) {
	if len(value) == 0 {
		return nil, nil
	}
	parts := make([]string, 0, len(value))
	for k, v := range value {
		if sk, ok := k.(string); ok {
			if sv, ok := v.(string); ok {
				parts = append(parts, sk+sep+sv)
			} else {
				return nil, fmt.Errorf("Cannot unmarshal '%v' of type %T into a string value", v, v)
			}
		} else {
			return nil, fmt.Errorf("Cannot unmarshal '%v' of type %T into a string value", k, k)
		}
	}
	return parts, nil
}

// the following code is from libcompose/yaml/types_yaml (Apache License)
func toStrings(s []interface{}) ([]string, error) {
	if len(s) == 0 {
		return nil, nil
	}
	r := make([]string, len(s))
	for k, v := range s {
		if sv, ok := v.(string); ok {
			r[k] = sv
		} else {
			return nil, fmt.Errorf("Cannot unmarshal '%v' of type %T into a string value", v, v)
		}
	}
	return r, nil
}
