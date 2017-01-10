package manifest

import "fmt"

type Protocol string
const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

type Port struct {
	Name      string
	Balancer  int
	Container int
	Protocol  Protocol
	Public    bool
}

func (p Port) String() string {
	if p.Public {
		return fmt.Sprintf("%d:%d/%s", p.Balancer, p.Container, string(p.Protocol))
	} else {
		return fmt.Sprintf("%d/%s", p.Container, string(p.Protocol))
	}
}

type Ports []Port

func (pp Ports) HasPublic() bool {
	for _, p := range pp {
		if p.Public {
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
		if p.Public {
			pp[i].Balancer += shift
		}
	}
}
