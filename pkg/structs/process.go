package structs

import (
	"fmt"
	"time"
)

type Process struct {
	Id string `json:"id"`

	App      string    `json:"app"`
	Command  string    `json:"command"`
	Cpu      float64   `json:"cpu"`
	Host     string    `json:"host"`
	Image    string    `json:"image"`
	Instance string    `json:"instance"`
	Memory   float64   `json:"memory"`
	Name     string    `json:"name"`
	Ports    []string  `json:"ports"`
	Release  string    `json:"release"`
	Started  time.Time `json:"started"`
	Status   string    `json:"status"`
}

type Processes []Process

type ProcessExecOptions struct {
	Entrypoint *bool `header:"Entrypoint"`
	Height     *int  `header:"Height"`
	Tty        *bool `header:"Tty" default:"true"`
	Width      *int  `header:"Width"`
}

type ProcessListOptions struct {
	Release *string `flag:"release" query:"release"`
	Service *string `flag:"service,s" query:"service"`
}

type ProcessRunOptions struct {
	Command     *string           `header:"Command"`
	Environment map[string]string `header:"Environment"`
	Height      *int              `header:"Height"`
	Image       *string           `header:"Image"`
	Memory      *int              `header:"Memory"`
	Release     *string           `flag:"release" header:"Release"`
	Volumes     map[string]string `header:"Volumes"`
	Width       *int              `header:"Width"`
}

func (p *Process) sortKey() string {
	return fmt.Sprintf("%s-%s", p.Name, p.Id)
}

func (ps Processes) Less(i, j int) bool {
	return ps[i].sortKey() < ps[j].sortKey()
}
