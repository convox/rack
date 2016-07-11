package manifest

import (
	"bufio"
	"bytes"
	"crypto/sha1"
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
	Image       string      `yaml:"image,omitempty"`
	Labels      Labels      `yaml:"labels,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Networks    []string    `yaml:"networks,omitempty"`
	Ports       Ports       `yaml:"ports,omitempty"`
	Privileged  bool        `yaml:"privileged,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`

	//TODO from models manifest, not passive and used at runtime
	Exports  map[string]string        `yaml:"-"`
	LinkVars map[string]template.HTML `yaml:"-"`

	randoms map[string]int
}

// see yaml.go for unmarshallers
type Build struct {
	Context    string
	Dockerfile string
	Args       map[string]string
}
type Command []string
type Environment map[string]string
type Labels map[string]string

func (s Service) HasBalancer() bool {
	return len(s.Ports) > 0
}

func (s *Service) Process(app string) Process {
	return NewProcess(app, *s)
}

func (s *Service) Proxies(app string) []Proxy {
	proxies := []Proxy{}

	for i, p := range s.Ports {
		if p.External() {
			name := fmt.Sprintf("%s-%s-proxy-%d", app, s.Name, p.Balancer)

			proxy := Proxy{
				Name:      name,
				Balancer:  p.Balancer,
				Container: p.Container,
				Host:      fmt.Sprintf("%s-%s", app, s.Name),
			}

			s.Ports[i].Balancer = 0

			proxy.Protocol = coalesce(s.Labels[fmt.Sprintf("convox.port.%s.protocol", p.Name)], "tcp")
			proxy.Proxy = s.Labels[fmt.Sprintf("convox.port.%s.proxy", p.Name)] == "true"
			proxy.Secure = s.Labels[fmt.Sprintf("convox.port.%s.secure", p.Name)] == "true"

			proxies = append(proxies, proxy)
		}
	}

	return proxies
}

func (s *Service) SyncPaths() (map[string]string, error) {
	sp := map[string]string{}

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
		case "ADD", "COPY":
			if len(parts) >= 3 {
				sp[parts[1]] = parts[2]
			}
		}
	}

	return sp, nil
}

func (s *Service) Tag() string {
	if s.Build.Context != "" {
		dockerFile := s.Build.Dockerfile
		if dockerFile == "" {
			dockerFile = s.Dockerfile
		}
		return tagHash(fmt.Sprintf("%s:%s", s.Build.Context, dockerFile))
	} else {
		return tagHash(s.Image)
	}
}

func containerEnv(container string) map[string]string {
	es := []string{}

	data, _ := Docker("inspect", "-f", "{{json .Config.Env}}", container).Output()

	json.Unmarshal(data, &es)

	env := map[string]string{}

	for _, e := range es {
		parts := strings.SplitN(e, "=", 2)
		env[parts[0]] = parts[1]
	}

	return env
}

func containerHost(container string) string {
	data, _ := Docker("inspect", "-f", "{{.NetworkSettings.IPAddress}}", container).Output()

	if s := strings.TrimSpace(string(data)); s != "" {
		return s
	}

	// TODO container is part of network, look up IP there

	return ""
}

func containerPort(container string) string {
	data, _ := Docker("inspect", "-f", "{{range $k,$v := .Config.ExposedPorts}}{{$k}}|{{end}}", container).Output()

	return strings.Split(string(data), "/")[0]
}

func linkArgs(name, container string) []string {
	args := []string{}

	prefix := strings.Replace(strings.ToUpper(name), "-", "_", -1)
	env := containerEnv(container)

	scheme := coalesce(env["LINK_SCHEME"], "tcp")
	host := containerHost(container)
	port := containerPort(container)
	path := env["LINK_PATH"]
	username := env["LINK_USERNAME"]
	password := env["LINK_PASSWORD"]

	args = append(args, "--add-host", fmt.Sprintf("%s:%s", name, containerHost(container)))
	args = append(args, "-e", fmt.Sprintf("%s_SCHEME=%s", prefix, scheme))
	args = append(args, "-e", fmt.Sprintf("%s_HOST=%s", prefix, host))
	args = append(args, "-e", fmt.Sprintf("%s_PORT=%s", prefix, port))
	args = append(args, "-e", fmt.Sprintf("%s_PATH=%s", prefix, path))
	args = append(args, "-e", fmt.Sprintf("%s_USERNAME=%s", prefix, username))
	args = append(args, "-e", fmt.Sprintf("%s_PASSWORD=%s", prefix, password))

	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}

	if username != "" || password != "" {
		u.User = url.UserPassword(username, password)
	}

	args = append(args, "-e", fmt.Sprintf("%s_URL=%s", prefix, u.String()))

	return args
}

func tagHash(id string) string {
	return fmt.Sprintf("convox-%s", fmt.Sprintf("%x", sha1.Sum([]byte(id)))[0:10])
}

func (s Service) LabelsByPrefix(prefix string) map[string]string {
	returnLabels := make(map[string]string)
	for k, v := range s.Labels {
		if strings.HasPrefix(k, prefix) {
			returnLabels[k] = v
		}
	}
	return returnLabels
}

func (me Service) ExternalPorts() []Port {
	ext := []Port{}

	for _, port := range me.Ports {
		if port.Balancer != 0 {
			ext = append(ext, port)
		}
	}

	return ext
}

func (me Service) InternalPorts() []Port {
	internal := []Port{}

	for _, port := range me.Ports {
		if port.Balancer == 0 {
			internal = append(internal, port)
		}
	}

	return internal
}

func (me Service) ContainerPorts() []string {
	ext := []string{}

	for _, port := range me.Ports {
		if port.Container != 0 {
			ext = append(ext, strconv.Itoa(port.Container))
		}
	}

	sort.Strings(ext)

	return ext
}

func (me Service) RegistryImage(appName, buildId string, outputs map[string]string) string {
	if registryId := outputs["RegistryId"]; registryId != "" {
		return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), outputs["RegistryRepository"], me.Name, buildId)
	}

	return fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), appName, me.Name, buildId)
}

func (me *Service) Randoms() map[string]int {
	if me.randoms != nil {
		return me.randoms
	}

	currentPort := 5000
	me.randoms = make(map[string]int)
	for _, port := range me.Ports {
		if ManifestRandomPorts {
			me.randoms[strconv.Itoa(port.Balancer)] = rand.Intn(62000) + 3000
		} else {
			me.randoms[strconv.Itoa(port.Balancer)] = currentPort
			currentPort += 1
		}
	}
	return me.randoms
}
