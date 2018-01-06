package structs

import (
	"fmt"
	"io"
	"time"
)

type Process struct {
	Id string `json:"id"`

	App      string   `json:"app"`
	Command  string   `json:"command"`
	Cpu      float64  `json:"cpu"`
	Host     string   `json:"host"`
	Image    string   `json:"image"`
	Instance string   `json:"instance"`
	Memory   float64  `json:"memory"`
	Name     string   `json:"name"`
	Ports    []string `json:"ports"`
	Release  string   `json:"release"`

	Started time.Time `json:"started"`
}

type Processes []Process

type ProcessExecOptions struct {
	Entrypoint *bool
	Height     *int
	Stream     io.ReadWriter
	Width      *int
}

type ProcessListOptions struct {
	Service string
}

type ProcessRunOptions struct {
	Command     *string
	Environment map[string]string
	Height      *int
	Image       *string
	Input       io.Reader
	Links       []string
	Memory      *int64
	Name        *string
	Output      io.Writer
	Ports       map[string]string
	Release     *string
	Service     *string
	Stream      io.ReadWriter
	Volumes     map[string]string
	Width       *int
}

func (p *Process) sortKey() string {
	return fmt.Sprintf("%s-%s", p.Name, p.Id)
}

func (ps Processes) Less(i, j int) bool {
	return ps[i].sortKey() < ps[j].sortKey()
}
