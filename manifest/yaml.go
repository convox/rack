package manifest

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/go-units"
)

// Parses as [balancer?]:[container]/[protocol?], where [balancer] and [protocol] are optional
var portMappingRegex = regexp.MustCompile(`(?i)^(?:(\d+):)?(\d+)(?:/(udp|tcp))?$`)

// MarshalYAML implements the Marshaller interface for the Manifest type
func (m Manifest) MarshalYAML() (interface{}, error) {
	m.Version = "2"
	return m, nil
}

// MarshalYAML implements the Marshaller interface for the Port type
func (p Port) MarshalYAML() (interface{}, error) {
	return p.String(), nil
}

// MarshalYAML implements the Marshaller interface for the Command type
func (c Command) MarshalYAML() (interface{}, error) {

	if c.String != "" {
		return c.String, nil

	} else if len(c.Array) > 0 {
		return c.Array, nil
	}

	return nil, nil
}

// MarshalYAML implements the Marshaller interface for the Environment type
func (ee Environment) MarshalYAML() (interface{}, error) {
	res := []string{}

	for _, e := range ee {
		if e.Needed && e.Value == "" {
			res = append(res, e.Name)
		} else {
			res = append(res, fmt.Sprintf("%s=%s", e.Name, e.Value))
		}
	}

	return res, nil
}

// UnmarshalYAML implements the Unmarshaller interface.
func (b *Build) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}

	if err := unmarshal(&v); err != nil {
		return err
	}

	switch v.(type) {
	case string:
		b.Context = v.(string)
	case map[interface{}]interface{}:
		for mapKey, mapValue := range v.(map[interface{}]interface{}) {
			switch mapKey {
			case "context":
				b.Context = mapValue.(string)
			case "dockerfile":
				b.Dockerfile = mapValue.(string)
			case "args":
				args := map[string]string{}

				switch t := mapValue.(type) {
				case map[interface{}]interface{}:
					for k, v := range t {
						var ks, vs string

						switch t := k.(type) {
						case string:
							ks = t
						case int:
							ks = strconv.Itoa(t)
						default:
							return fmt.Errorf("unknown type in environment map: %v", k)
						}

						switch t := v.(type) {
						case string:
							vs = t
						case int:
							vs = strconv.Itoa(t)
						default:
							return fmt.Errorf("unknown type in environment map: %v", k)
						}

						args[ks] = vs
					}
				case []interface{}:
					for _, tt := range t {
						s, ok := tt.(string)

						if !ok {
							return fmt.Errorf("unknown type in environment list: %v", t)
						}

						parts := strings.SplitN(s, "=", 2)

						switch len(parts) {
						case 1:
							args[parts[0]] = ""
						case 2:
							args[parts[0]] = parts[1]
						default:
							return fmt.Errorf("cannot parse environment: %v", t)
						}
					}
				default:
					return fmt.Errorf("unknown type for args: %T", t)
				}

				b.Args = args
			default:
				// Ignore
				// unknown
				// keys
				continue
			}
		}
	default:
		return fmt.Errorf("Failed to unmarshal Build: %#v", v)
	}
	return nil
}

func (c *Command) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}

	if err := unmarshal(&v); err != nil {
		return err
	}

	switch t := v.(type) {
	case string:
		c.String = t
	case []interface{}:
		for _, tt := range t {
			s, ok := tt.(string)

			if !ok {
				return fmt.Errorf("unknown type in command array: %v", t)
			}

			c.Array = append(c.Array, s)
		}
	default:
		return fmt.Errorf("cannot parse command: %s", t)
	}

	return nil
}

func (e *Environment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}

	if err := unmarshal(&v); err != nil {
		return err
	}

	*e = make(Environment, 0)

	switch t := v.(type) {
	case map[interface{}]interface{}:
		for k, v := range t {
			var ks, vs string

			switch t := k.(type) {
			case string:
				ks = t
			case int:
				ks = strconv.Itoa(t)
			default:
				return fmt.Errorf("unknown type in environment map: %v", k)
			}

			switch t := v.(type) {
			case string:
				vs = t
			case int:
				vs = strconv.Itoa(t)
			default:
				return fmt.Errorf("unknown type in environment map: %v", k)
			}

			*e = append(*e, EnvironmentItem{
				Name:   ks,
				Value:  vs,
				Needed: false,
			})
		}
	case []interface{}:
		for _, tt := range t {
			s, ok := tt.(string)

			if !ok {
				return fmt.Errorf("unknown type in environment list: %v", t)
			}

			parts := strings.SplitN(s, "=", 2)

			switch len(parts) {
			case 1:
				*e = append(*e, EnvironmentItem{
					Name:   parts[0],
					Needed: true,
				})
			case 2:
				*e = append(*e, EnvironmentItem{
					Name:  parts[0],
					Value: parts[1],
				})
			default:
				return fmt.Errorf("cannot parse environment: %v", t)
			}
		}
	default:
		return fmt.Errorf("cannot parse environment: %v", t)
	}

	sort.Sort(*e)

	return nil
}

func (l *Labels) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}

	if err := unmarshal(&v); err != nil {
		return err
	}

	*l = make(Labels)

	switch t := v.(type) {
	case map[interface{}]interface{}:
		for k, v := range t {
			var ks, vs string

			switch t := k.(type) {
			case string:
				ks = t
			case int:
				ks = strconv.Itoa(t)
			default:
				return fmt.Errorf("unknown type in label map: %v", k)
			}

			switch t := v.(type) {
			case string:
				vs = t
			case int:
				vs = strconv.Itoa(t)
			case bool:
				vs = strconv.FormatBool(t)
			default:
				return fmt.Errorf("unknown type in label map: %v", k)
			}

			(*l)[ks] = vs
		}
	case []interface{}:
		for _, tt := range t {
			s, ok := tt.(string)

			if !ok {
				return fmt.Errorf("unknown type in command array: %v", t)
			}

			parts := strings.SplitN(s, "=", 2)

			switch len(parts) {
			case 2:
				if _, ok := (*l)[parts[0]]; ok {
					return fmt.Errorf("duplicate cron label %v", parts[0])
				}
				(*l)[parts[0]] = parts[1]
			default:
				return fmt.Errorf("cannot parse label: %v", t)
			}
		}
	default:
		return fmt.Errorf("cannot parse labels: %v", t)
	}

	return nil
}

func (m *Memory) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}

	if err := unmarshal(&v); err != nil {
		return err
	}

	switch t := v.(type) {
	case string:
		ram, err := units.RAMInBytes(t)
		if err != nil {
			return err
		}
		*m = Memory(ram)
	case int:
		*m = Memory(t)
	case float64:
		*m = Memory(t)
	default:
		return fmt.Errorf("could not parse mem_limit: %v", t)
	}

	return nil
}

func (pp *Ports) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v []string

	if err := unmarshal(&v); err != nil {
		return err
	}

	*pp = make(Ports, len(v))

	for i, s := range v {
		parts := portMappingRegex.FindStringSubmatch(s)
		if len(parts) == 0 {
			return fmt.Errorf("invalid portmapping %s", s)
		}

		p := Port{}
		p.Name = parts[1]

		protocol := strings.ToLower(parts[3])
		if protocol != string(TCP) && protocol != string(UDP) {
			protocol = string(TCP) // default
		}
		p.Protocol = Protocol(protocol)

		container, _ := strconv.Atoi(parts[2])
		p.Container = container
		p.Balancer = container

		if parts[1] != "" {
			balancer, _ := strconv.Atoi(parts[1])
			p.Balancer = balancer
		}

		// Only TCP ports can be "public" (in the ELB sense) or have an ELB at all
		if parts[1] != "" && p.Protocol == TCP {
			p.Public = true
		}

		(*pp)[i] = p
	}

	return nil
}
