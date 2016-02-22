package structs

import (
	"fmt"
	"time"
)

type Process struct {
	Id        string    `json:"id"`
	App       string    `json:"app"`
	Binds     []string  `json:"binds"`
	Command   string    `json:"command"`
	Container string    `json:"container"`
	Cpu       float64   `json:"cpu"`
	Host      string    `json:"host"`
	Image     string    `json:"image"`
	Memory    float64   `json:"memory"`
	Name      string    `json:"name"`
	Ports     []string  `json:"ports"`
	Release   string    `json:"release"`
	Size      int64     `json:"size"`
	Started   time.Time `json:"started"`
}

type Processes []Process

type ProcessStats struct {
	Cpu    float64
	Memory float64
}

func (ps Processes) Len() int {
	return len(ps)
}

// Sort processes by name and id
// Processes with a 'pending' id will naturally come last by design
func (ps Processes) Less(i, j int) bool {
	psi := fmt.Sprintf("%s-%s", ps[i].Name, ps[i].Id)
	psj := fmt.Sprintf("%s-%s", ps[j].Name, ps[j].Id)

	return psi < psj
}

func (ps Processes) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}
