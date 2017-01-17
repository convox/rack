package manifest

import "fmt"

// A Protocol represents a network protocol type (TCP, UDP, etc)
type Protocol string
const (
	// TCP network protocol
	TCP Protocol = "tcp"

	// UDP network protocol
	UDP Protocol = "udp"
)

// A Port represents a port as defined in docker-compose.yml
// Currently, UDP ports cannot be Public, as AWS ELBs only support TCP
type Port struct {
	Name      string
	Balancer  int       // the port exposed by the load balancer for TCP, or by the host for UDP
	Container int       // the port used in the container and exposed by the Dockerfile
	Protocol  Protocol  // the network protocol used (ie, TCP or UDP)
	Public    bool      // whether the port is internet-facing or internal-only on the load balancer (TCP only)
}

// String returns a string representation of a Port struct
func (p Port) String() string {
	if p.Public {
		return fmt.Sprintf("%d:%d/%s", p.Balancer, p.Container, string(p.Protocol))
	}

	return fmt.Sprintf("%d/%s", p.Container, string(p.Protocol))
}

// Ports is a collection of Port structs
type Ports []Port

// HasPublic returns true if any Port in the collection is Public
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
