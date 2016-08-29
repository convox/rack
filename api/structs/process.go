package structs

import (
	"fmt"
	"time"
)

type Process struct {
	ID      string `json:"id"`
	App     string `json:"app"`
	Name    string `json:"name"`
	Release string `json:"release"`

	Command  string   `json:"command"`
	Host     string   `json:"host"`
	Image    string   `json:"image"`
	Instance string   `json:"instance"`
	Ports    []string `json:"ports"`

	Cpu    float64 `json:"cpu"`
	Memory float64 `json:"memory"`

	Started time.Time `json:"started"`
}

type Processes []Process

type ProcessRunOptions struct {
	Command string
	Release string
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
