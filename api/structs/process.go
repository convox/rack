package structs

import (
	"fmt"
	"io"
	"time"
)

// Process represents a running Process
type Process struct {
	ID      string `json:"id"`
	App     string `json:"app"`
	Name    string `json:"name"`
	Group   string `json:"group"`
	Release string `json:"release"`

	Command  string   `json:"command"`
	Host     string   `json:"host"`
	Image    string   `json:"image"`
	Instance string   `json:"instance"`
	Ports    []string `json:"ports"`

	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`

	Started time.Time `json:"started"`
}

// Processes are a list of Processes
type Processes []Process

// ProcessExecOptions are options for ProcessExec
type ProcessExecOptions struct {
	Height int
	Width  int
}

// ProcessRunOptions are options for ProcessRun
type ProcessRunOptions struct {
	Command string
	Height  int
	Width   int
	Release string
	Stream  io.ReadWriter
}

func (ps Processes) Len() int {
	return len(ps)
}

// Sort processes by name and id
// Processes with a 'pending' id will naturally come last by design
func (ps Processes) Less(i, j int) bool {
	psi := fmt.Sprintf("%s-%s", ps[i].Name, ps[i].ID)
	psj := fmt.Sprintf("%s-%s", ps[j].Name, ps[j].ID)

	return psi < psj
}

func (ps Processes) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}
