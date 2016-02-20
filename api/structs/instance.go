package structs

import "time"

type Instance struct {
	Agent     bool      `json:"agent"`
	Cpu       float64   `json:"cpu"`
	Id        string    `json:"id"`
	Ip        string    `json:"ip"`
	Memory    float64   `json:"memory"`
	Processes int       `json:"processes"`
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
