package manifest

import "fmt"

type Proxy struct {
	Name string

	Balancer  int
	Container int

	Protocol string
	Host     string
	Proxy    bool
	Secure   bool
}

func (p *Proxy) Start() error {
	Docker("rm", "-f", p.Name).Run()

	args := []string{"run"}

	args = append(args, "--rm", "--name", p.Name)
	args = append(args, "-p", fmt.Sprintf("%d:%d", p.Balancer, p.Balancer))
	args = append(args, "--link", fmt.Sprintf("%s:host", p.Host))
	args = append(args, "convox/proxy", fmt.Sprintf("%d", p.Balancer), fmt.Sprintf("%d", p.Container), p.Protocol)

	if p.Proxy {
		args = append(args, "proxy")
	}

	if p.Secure {
		args = append(args, "secure")
	}

	cmd := Docker(args...)

	err := cmd.Start()

	go cmd.Wait()

	return err
}
