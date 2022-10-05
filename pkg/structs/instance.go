package structs

import (
	"os"
	"time"
)

type Instance struct {
	Agent     bool      `json:"agent"`
	Cpu       float64   `json:"cpu"`
	Id        string    `json:"id"`
	Memory    float64   `json:"memory"`
	PrivateIp string    `json:"private-ip"`
	Processes int       `json:"processes"`
	PublicIp  string    `json:"public-ip"`
	Status    string    `json:"status"`
	Started   time.Time `json:"started"`
}

type Instances []Instance

type InstanceShellOptions struct {
	Command *string `header:"Command"`
	Height  *int    `header:"Height"`
	Width   *int    `header:"Width"`
}

func (i *Instance) Ip() string {
	if os.Getenv("DEVELOPMENT") == "true" {
		return i.PublicIp
	}

	return i.PrivateIp
}

func (ii Instances) Len() int           { return len(ii) }
func (ii Instances) Less(i, j int) bool { return ii[i].Id < ii[j].Id }
func (ii Instances) Swap(i, j int)      { ii[i], ii[j] = ii[j], ii[i] }
