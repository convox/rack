package manifest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var ManifestRandomPorts = true

type Service struct {
	Name string `yaml:"-"`

	Build       Build       `yaml:"build,omitempty"`
	Command     Command     `yaml:"command,omitempty"`
	Dockerfile  string      `yaml:"dockerfile,omitempty"`
	Entrypoint  string      `yaml:"entrypoint,omitempty"`
	Environment Environment `yaml:"environment,omitempty"`
	ExtraHosts  []string    `yaml:"extra_hosts,omitempty"`
	Image       string      `yaml:"image,omitempty"`
	Labels      Labels      `yaml:"labels,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Networks    Networks    `yaml:"-"`
	Ports       Ports       `yaml:"ports,omitempty"`
	Privileged  bool        `yaml:"privileged,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`

	Cpu    int64  `yaml:"cpu_shares,omitempty"`
	Memory Memory `yaml:"mem_limit,omitempty"`

	//TODO from models manifest, not passive and used at runtime
	Exports  map[string]string        `yaml:"-"`
	LinkVars map[string]template.HTML `yaml:"-"`

	Primary bool `yaml:"-"`

	randoms map[string]int
}

// Services are a list of Services
type Services []Service

// see yaml.go for unmarshallers
type Build struct {
	Context    string            `yaml:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
}

// Command represents the command docker will run either in string or array of strings
type Command struct {
	String string   `yaml:"-"`
	Array  []string `yaml:"-"`
}

// EnvironmentItem is a single item in an environment
type EnvironmentItem struct {
	Name   string
	Value  string
	Needed bool // if true this value should be provided by the user in .env or convox env
}

// Environment is a service's default environment
type Environment []EnvironmentItem

type Labels map[string]string
type Memory int64
type Networks map[string]InternalNetwork

type InternalNetwork map[string]ExternalNetwork
type ExternalNetwork Network

type Network struct {
	Name string
}

// Hash returns a string suitable for using as a map key
func (b *Build) Hash() string {
	argKeys := []string{}
	for k := range b.Args {
		argKeys = append(argKeys, k)
	}
	sort.Strings(argKeys)

	hashParts := make([]string, len(argKeys))
	for i, key := range argKeys {
		hashParts[i] = fmt.Sprintf("%s=%s", key, b.Args[key])
	}
	argsHash := strings.Join(hashParts, "@@@@@")

	return fmt.Sprintf("%+v|||||%+v|||||%+v", b.Context, b.Dockerfile, argsHash)
}

func (s *Service) Process(app string, m Manifest) Process {
	return NewProcess(app, *s, m)
}

// UseSecureEnvironment - Determines if the service intends to use a secure environment
func (s Service) UseSecureEnvironment() bool {
	return s.Labels["convox.environment.secure"] == "true"
}

// HasBalancer returns false if the Service contains no public ports,
// or if the `convox.balancer` label is set to false
func (s Service) HasBalancer() bool {
	if s.Labels["convox.balancer"] == "false" {
		return false
	}

	for _, p := range s.Ports {
		if p.Protocol == TCP {
			return true
		}
	}

	return false
}

// IsAgent returns true if Service is a per-host agent, otherwise it returns false
func (s Service) IsAgent() bool {
	// BackCompat: Some people might already be trying to use `convox.daemon` based on
	// https://github.com/convox-examples/dd-agent/blob/master/docker-compose.yml#L13
	_, okAgent := s.Labels["convox.agent"]
	_, okDaemon := s.Labels["convox.daemon"]
	return okAgent || okDaemon
}

// DefaultParams returns a string of comma-delimited Count, CPU, and Memory params
func (s Service) DefaultParams() string {
	count := 1
	cpu := 0
	memory := 256

	if s.IsAgent() {
		count = 0
	}

	return fmt.Sprintf("%d,%d,%d", count, cpu, memory)
}

func (s *Service) Proxies(app string) []Proxy {
	proxies := []Proxy{}

	for i, p := range s.Ports {
		if p.Public {
			name := fmt.Sprintf("%s-%s-proxy-%d", app, s.Name, p.Balancer)

			proxy := Proxy{
				Name:      name,
				Balancer:  p.Balancer,
				Container: p.Container,
				Host:      fmt.Sprintf("%s-%s", app, s.Name),
				Network:   s.NetworkName(),
			}

			s.Ports[i].Balancer = 0

			proxy.Protocol = coalesce(s.Labels[fmt.Sprintf("convox.port.%d.protocol", p.Balancer)], "tcp")
			proxy.Proxy = s.Labels[fmt.Sprintf("convox.port.%d.proxy", p.Balancer)] == "true"
			proxy.Secure = s.Labels[fmt.Sprintf("convox.port.%d.secure", p.Balancer)] == "true"

			proxies = append(proxies, proxy)
		}
	}

	return proxies
}

func (s *Service) SyncPaths() (map[string]string, error) {
	sp := map[string]string{}
	ev := map[string]string{}

	if s.Build.Context == "" {
		return sp, nil
	}

	dockerFile := s.Build.Dockerfile
	if dockerFile == "" {
		dockerFile = s.Dockerfile
	}

	data, err := ioutil.ReadFile(filepath.Join(s.Build.Context, coalesce(dockerFile, "Dockerfile")))
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 1 {
			continue
		}

		switch parts[0] {
		case "ENV":
			if len(parts) >= 3 {
				ev[parts[1]] = parts[2]
			}
		case "ADD", "COPY":
			if len(parts) >= 3 {
				u, err := url.Parse(parts[1])
				if err != nil {
					return nil, err
				}

				switch u.Scheme {
				case "http", "https":
					// do nothing
				default:
					path := parts[2]
					for k, v := range ev {
						if strings.Contains(path, fmt.Sprintf("$%s", k)) {
							path = strings.Replace(path, fmt.Sprintf("$%s", k), v, -1)
						}
					}
					sp[filepath.Join(s.Build.Context, parts[1])] = path
				}
			}
		}
	}

	return sp, nil
}

// Tag generates a string used to tag an image.
func (s *Service) Tag(appName string) string {
	return (fmt.Sprintf("%s/%s", appName, strings.Replace(s.Name, "_", "-", -1)))
}

// MountableVolume describes a mountable volume
type MountableVolume struct {
	Host      string
	Container string
}

// MountableVolumes return the mountable volumes for a service
func (s Service) MountableVolumes() []MountableVolume {
	volumes := []MountableVolume{}

	for _, volume := range s.Volumes {
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

// IsSystem white lists special host volumes to pass through to the container
// instead of turning them into an application EFS mount
func (v MountableVolume) IsSystem() bool {
	switch v.Host {
	case "/var/run/docker.sock":
		return true
	case "/proc/":
		return true
	case "/cgroup/":
		return true
	default:
		return false
	}
}

// DeploymentMinimum returns the min percent of containers that are allowed during deployment
func (s Service) DeploymentMinimum() string {
	return s.LabelDefault("convox.deployment.minimum", "100")
}

// DeploymentMaximum returns the max percent of containers that are allowed during deployment
// This will be most likely be overridden and set to 100 for singleton processes like schedulers that cannot have 2 running at once
func (s Service) DeploymentMaximum() string {
	return s.LabelDefault("convox.deployment.maximum", "200")
}

// NetworkName returns custom network name from the networks, defined in compose file.
// REturns empty string, if no custom network is defined.
// We pick the last one, as we currently support only single one.
func (s *Service) NetworkName() string {
	// No custom docker network by default
	networkName := ""

	for _, n := range s.Networks {
		for _, in := range n {
			networkName = in.Name
		}
	}
	return networkName
}

func containerEnv(container string) map[string]string {
	es := []string{}

	data, _ := Docker("inspect", "-f", "{{json .Config.Env}}", container).Output()

	json.Unmarshal(data, &es)

	env := map[string]string{}

	for _, e := range es {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) > 1 {
			env[parts[0]] = parts[1]
		} else {
			env[parts[0]] = ""
		}
	}

	return env
}

func containerHost(container string, networks Networks) string {
	ipFilterString := "{{ .NetworkSettings.IPAddress }}"

	// TODO container is part of network, look up IP there
	if len(networks) > 0 {
		for _, n := range networks {
			for _, in := range n {
				ipFilterString = "{{ .NetworkSettings.Networks." + in.Name + ".IPAddress }}"
				break
			}
			break
		}
	}

	data, _ := Docker("inspect", "-f", ipFilterString, container).Output()

	if s := strings.TrimSpace(string(data)); s != "" {
		return s
	}

	return ""
}

func containerPort(container string) string {
	data, _ := Docker("inspect", "-f", "{{range $k,$v := .Config.ExposedPorts}}{{$k}}|{{end}}", container).Output()

	return strings.Split(string(data), "/")[0]
}

func linkArgs(s Service, container string) []string {
	args := []string{}

	prefix := strings.Replace(strings.ToUpper(s.Name), "-", "_", -1)
	env := containerEnv(container)

	scheme := coalesce(env["LINK_SCHEME"], "tcp")
	host := containerHost(container, s.Networks)
	port := coalesce(env["LINK_PORT"], containerPort(container))
	path := env["LINK_PATH"]
	username := env["LINK_USERNAME"]
	password := env["LINK_PASSWORD"]

	args = append(args, "--add-host", fmt.Sprintf("%s:%s", s.Name, host))
	args = append(args, "-e", fmt.Sprintf("%s_SCHEME=%s", prefix, scheme))
	args = append(args, "-e", fmt.Sprintf("%s_HOST=%s", prefix, host))
	args = append(args, "-e", fmt.Sprintf("%s_PORT=%s", prefix, port))
	args = append(args, "-e", fmt.Sprintf("%s_PATH=%s", prefix, path))
	args = append(args, "-e", fmt.Sprintf("%s_USERNAME=%s", prefix, username))
	args = append(args, "-e", fmt.Sprintf("%s_PASSWORD=%s", prefix, password))

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   path,
	}

	if username != "" || password != "" {
		u.User = url.UserPassword(username, password)
	}

	args = append(args, "-e", fmt.Sprintf("%s_URL=%s", prefix, u.String()))

	return args
}

func (ee Environment) Len() int           { return len(ee) }
func (ee Environment) Less(i, j int) bool { return ee[i].Name < ee[j].Name }
func (ee Environment) Swap(i, j int)      { ee[i], ee[j] = ee[j], ee[i] }

// LabelsByPrefix retuns a map of string values with the labels filtered by prefix
func (s Service) LabelsByPrefix(prefix string) map[string]string {
	returnLabels := make(map[string]string)
	for k, v := range s.Labels {
		if strings.HasPrefix(k, prefix) {
			returnLabels[k] = v
		}
	}
	return returnLabels
}

// LabelDefault returns the value of a given label if it exists, otherwise the specified default
func (s Service) LabelDefault(label, def string) string {
	if val, ok := s.Labels[label]; ok {
		return val
	}

	return def
}

// ExternalPorts returns a collection of Port structs from the Service which are TCP and Public
func (s Service) ExternalPorts() []Port {
	ports := []Port{}

	for _, port := range s.Ports {
		if port.Public && port.Protocol == TCP {
			ports = append(ports, port)
		}
	}

	return ports
}

// InternalPorts returns a collection of Port structs from the Service which are TCP and not Public
func (s Service) InternalPorts() []Port {
	ports := []Port{}

	for _, port := range s.Ports {
		if !port.Public && port.Protocol == TCP {
			ports = append(ports, port)
		}
	}

	return ports
}

// TCPPorts returns a collection of Port structs from the Service which are TCP
func (s Service) TCPPorts() []Port {
	ports := []Port{}

	for _, port := range s.Ports {
		if port.Protocol == TCP {
			ports = append(ports, port)
		}
	}

	return ports
}

// UDPPorts returns a collection of Port structs from the Service which are UDP
func (s Service) UDPPorts() []Port {
	ports := []Port{}

	for _, port := range s.Ports {
		if port.Protocol == UDP {
			ports = append(ports, port)
		}
	}

	return ports
}

func (s Service) ContainerPorts() []string {
	ext := []string{}

	for _, port := range s.Ports {
		if port.Container != 0 {
			ext = append(ext, strconv.Itoa(port.Container))
		}
	}

	sort.Strings(ext)

	return ext
}

func (s Service) ParamName(name string) string {
	return fmt.Sprintf("%s%s", UpperName(s.Name), name)
}

func (s Service) RegistryImage(appName, buildId string, outputs map[string]string) string {
	if registryId := outputs["RegistryId"]; registryId != "" {
		return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), outputs["RegistryRepository"], s.Name, buildId)
	}

	return fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), appName, s.Name, buildId)
}

//ExtraHostsMap is a convenience method to allow for easier use of the hosts in
//AWS templates
func (s Service) ExtraHostsMap() map[string]string {
	res := map[string]string{}
	for _, str := range s.ExtraHosts {
		parts := strings.Split(str, ":")
		res[parts[0]] = parts[1]
	}
	return res
}

func (s *Service) Randoms() map[string]int {
	if s.randoms != nil {
		return s.randoms
	}

	currentPort := 5000
	s.randoms = make(map[string]int)
	for _, port := range s.Ports {
		if ManifestRandomPorts {
			s.randoms[strconv.Itoa(port.Balancer)] = rand.Intn(62000) + 3000
		} else {
			s.randoms[strconv.Itoa(port.Balancer)] = currentPort
			currentPort += 1
		}
	}
	return s.randoms
}

func (ss Services) Len() int {
	return len(ss)
}

func (ss Services) Less(i, j int) bool {
	return ss[i].Name < ss[j].Name
}

func (ss Services) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}
