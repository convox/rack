package manifest

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

func (m *Manifest) MarshalYAML() (interface{}, error) {
	log.Print("MARSHALYAML CALLED")
	if m.Version == "1" {
		return m.Services, nil
	}
	return map[string]interface{}{
		"Version":  m.Version,
		"Services": m.Services,
	}, nil
}

func (p *Port) MarshalYAML() (interface{}, error) {
	if p.Public {
		return fmt.Sprintf("%d:%d", p.Balancer, p.Container), nil
	}
	return fmt.Sprintf("%d", p.Container), nil
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
				//TODO
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
		*c = []string{"sh", "-c", t}
	case []interface{}:
		for _, tt := range t {
			s, ok := tt.(string)

			if !ok {
				return fmt.Errorf("unknown type in command array: %v", t)
			}

			*c = append(*c, s)
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

	*e = make(Environment)

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
			default:
				return fmt.Errorf("unknown type in label map: %v", k)
			}

			(*e)[ks] = vs
		}
	case []interface{}:
		for _, tt := range t {
			s, ok := tt.(string)

			if !ok {
				return fmt.Errorf("unknown type in command array: %v", t)
			}

			parts := strings.SplitN(s, "=", 2)

			switch len(parts) {
			case 1:
				(*e)[parts[0]] = ""
			case 2:
				(*e)[parts[0]] = parts[1]
			default:
				return fmt.Errorf("cannot parse environment: %v", t)
			}
		}
	default:
		return fmt.Errorf("cannot parse environment: %v", t)
	}

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

func (pp *Ports) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v []string

	if err := unmarshal(&v); err != nil {
		return err
	}

	*pp = make(Ports, len(v))

	for i, s := range v {
		parts := strings.Split(s, ":")
		p := Port{}

		switch len(parts) {
		case 1:
			n, err := strconv.Atoi(parts[0])

			if err != nil {
				return fmt.Errorf("error parsing port: %s", err)
			}

			p.Name = parts[0]
			p.Container = n
			p.Balancer = n
			p.Public = false
		case 2:
			n, err := strconv.Atoi(parts[0])

			if err != nil {
				return fmt.Errorf("error parsing port: %s", err)
			}

			p.Balancer = n

			n, err = strconv.Atoi(parts[1])

			if err != nil {
				return fmt.Errorf("error parsing port: %s", err)
			}

			p.Name = parts[0]
			p.Container = n
			p.Public = true
		default:
			return fmt.Errorf("invalid port: %s", s)
		}

		(*pp)[i] = p
	}

	return nil
}
