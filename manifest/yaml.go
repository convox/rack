package manifest

import (
	"fmt"
	"strconv"
	"strings"
)

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
