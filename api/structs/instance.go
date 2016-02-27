package structs

import (
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Instance struct {
	Agent     bool      `json:"agent"`
	Cpu       float64   `json:"cpu"`
	Id        string    `json:"id"`
	Memory    float64   `json:"memory"`
	PrivateIp string    `json:"private_ip"`
	Processes int       `json:"processes"`
	PublicIp  string    `json:"public_ip"`
	Status    string    `json:"status"`
	Started   time.Time `json:"started"`
}

type Instances []Instance

type InstanceResource struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

func (ir InstanceResource) PercentUsed() float64 {
	return float64(ir.Used) / float64(ir.Total)
}

func (i *Instance) DockerClient() (*docker.Client, error) {
	if os.Getenv("DEVELOPMENT") == "true" {
		return docker.NewClient(i.PublicIp)
	}

	return docker.NewClient(i.PrivateIp)
}
