package manifest

import "fmt"

type Port struct {
	Name      string
	Balancer  int
	Container int
	Public    bool
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

func (p Port) External() bool {
	return p.Public
}

func (p Port) String() string {
	if p.External() {
		return fmt.Sprintf("%d:%d", p.Balancer, p.Container)
	} else {
		return fmt.Sprintf("%d", p.Container)
	}
}
