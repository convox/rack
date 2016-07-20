package manifest

import (
	"fmt"
	"strconv"
	"strings"
)

type Port struct {
	Name      string
	Balancer  int
	Container int
}

type Ports []Port

func (pp Ports) External() bool {
	for _, p := range pp {
		if p.External() {
			return true
		}
	}

	return false
}

// func (pp Ports) StartProxies() error {
//   for _, p := range pp {
//     if err := pp.StartProxy(); err != nil {
//       return err
//     }
//   }

//   return nil
// }

// Shift all external ports by the given amount
//
// If it's an internal-only port then make it external before incrementing
func (pp Ports) Shift(shift int) {
	for i, p := range pp {
		if p.External() {
			pp[i].Balancer += shift
		}
	}
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
		default:
			return fmt.Errorf("invalid port: %s", s)
		}

		(*pp)[i] = p
	}

	return nil
}

func (p Port) External() bool {
	return p.Balancer != 0
}

func (p Port) String() string {
	if p.External() {
		return fmt.Sprintf("%d:%d", p.Balancer, p.Container)
	} else {
		return fmt.Sprintf("%d", p.Container)
	}
}
