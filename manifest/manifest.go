package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var interpolationBracketRegex = regexp.MustCompile("\\$\\{([0-9A-Za-z_]*)\\}")
var interpolationDollarRegex = regexp.MustCompile("\\$([0-9A-Za-z_]+)")

type Manifest struct {
	Version  string             `yaml:"version"`
	Networks Networks           `yaml:"networks,omitempty"`
	Services map[string]Service `yaml:"services"`
}

// Load a Manifest from raw data
func Load(data []byte) (*Manifest, error) {
	data, err := parseEnvVars(data)
	if err != nil {
		return nil, err
	}

	v, err := manifestVersion(data)
	if err != nil {
		return nil, err
	}

	m := &Manifest{Version: v}

	switch v {
	case "1":
		if err := yaml.Unmarshal(data, &m.Services); err != nil {
			return nil, fmt.Errorf("error loading manifest: %s", err)
		}
	case "2":
		if err := yaml.Unmarshal(data, m); err != nil {
			return nil, fmt.Errorf("error loading manifest: %s", err)
		}
	default:
		return nil, fmt.Errorf("unknown manifest version: %s", v)
	}

	for name, service := range m.Services {
		service.Name = name

		// there are two places in a docker-compose.yml to specify a dockerfile
		// normalize (for caching) and complain if both are set
		if service.Dockerfile != "" {
			if service.Build.Dockerfile != "" {
				return nil, fmt.Errorf("dockerfile specified twice for %s", name)
			}
			service.Build.Dockerfile = service.Dockerfile
		}

		// denormalize a bit
		service.Networks = m.Networks

		m.Services[name] = service
	}

	err = m.Validate()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Load a Manifest from a file
func LoadFile(path string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return Load(data)
}

func (m Manifest) Validate() error {
	regexValidCronLabel := regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)

	for _, entry := range m.Services {
		if strings.Contains(entry.Name, "_") {
			return fmt.Errorf("service name cannot contain an underscore: %s", entry.Name)
		}

		labels := entry.LabelsByPrefix("convox.cron")
		for k, _ := range labels {
			parts := strings.Split(k, ".")
			if len(parts) != 3 {
				return fmt.Errorf(
					"Cron task is not valid (must be in format convox.cron.myjob)",
				)
			}
			name := parts[2]
			if !regexValidCronLabel.MatchString(name) {
				return fmt.Errorf(
					"Cron task %s is not valid (cron names can contain only alphanumeric characters and dashes and must be between 4 and 30 characters)",
					name,
				)
			}
		}

		labels = entry.LabelsByPrefix("convox.health.timeout")
		for _, v := range labels {
			i, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf(
					"convox.health.timeout is invalid for %s, must be a number between 0 and 60",
					entry.Name,
				)
			}

			if i < 0 || i > 60 {
				return fmt.Errorf(
					"convox.health.timeout is invalid for %s, must be a number between 0 and 60",
					entry.Name,
				)
			}
		}

		for _, l := range entry.Links {
			ls, ok := m.Services[l]
			if !ok {
				return fmt.Errorf(
					"%s links to service: %s which does not exist",
					entry.Name,
					l,
				)
			}

			if len(ls.Ports) == 0 {
				return fmt.Errorf(
					"%s links to service: %s which does not expose any ports",
					entry.Name,
					l,
				)
			}
		}
	}
	return nil
}

// Return a list of ports this manifest will expose when run
func (m *Manifest) ExternalPorts() []int {
	ext := []int{}

	for _, service := range m.Services {
		for _, port := range service.Ports {
			if port.External() {
				ext = append(ext, port.Balancer)
			}
		}
	}

	return ext
}

// Find any port conflits that would prevent this manifest from running
func (m *Manifest) PortConflicts() ([]int, error) {
	ext := m.ExternalPorts()

	conflicts := make([]int, 0)

	host := dockerHost()

	for _, p := range ext {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, p), 200*time.Millisecond)

		if err == nil {
			conflicts = append(conflicts, p)
			defer conn.Close()
		}
	}

	sort.Ints(conflicts)

	return conflicts, nil
}

// Run Instantiate a Run object based on this manifest to be run via 'convox start'
func (m *Manifest) Run(dir, app string, cache, sync bool) Run {
	return NewRun(dir, app, *m, cache, sync)
}

// Return the Services of this Manifest in the order you should run them
func (m *Manifest) runOrder() Services {
	services := Services{}

	for _, service := range m.Services {
		services = append(services, service)
	}

	sort.Sort(services)

	// classic bubble sort
	for i := 0; i < len(services)-1; i++ {
		for j := i + 1; j < len(services); j++ {
			// swap if j is a dependency of i
			for _, name := range services[i].Links {
				if name == services[j].Name {
					services[i], services[j] = services[j], services[i]
					break
				}
			}
		}
	}

	return services
}

// Shift all external ports in this Manifest by the given amount and their shift labels
func (m *Manifest) Shift(shift int) error {
	for _, service := range m.Services {
		service.Ports.Shift(shift)

		if ss, ok := service.Labels["convox.start.shift"]; ok {
			shift, err := strconv.Atoi(ss)
			if err != nil {
				return fmt.Errorf("invalid shift: %s", ss)
			}

			service.Ports.Shift(shift)
		}
	}

	return nil
}

func manifestVersion(data []byte) (string, error) {
	var check struct {
		Version string
	}

	if err := yaml.Unmarshal(data, &check); err != nil {
		return "", fmt.Errorf("could not parse manifest: %s", err)
	}

	if check.Version != "" {
		return check.Version, nil
	}

	return "1", nil
}

func parseEnvVars(data []byte) ([]byte, error) {
	r := bytes.NewReader(data)
	result := []byte{}
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return result, err
		}
		result = append(result, []byte(parseLine(line))...)
		if err == io.EOF {
			break
		}
	}
	return result, nil
}

func (m *Manifest) Raw() ([]byte, error) {
	return yaml.Marshal(m)
}

func (m Manifest) EntryNames() []string {
	names := make([]string, len(m.Services))
	x := 0

	for k, _ := range m.Services {
		names[x] = k
		x += 1
	}

	return names
}

func (m Manifest) BalancerResourceName(process string) string {
	for _, b := range m.Balancers() {
		if b.Entry.Name == process {
			return b.ResourceName()
		}
	}

	return ""
}
