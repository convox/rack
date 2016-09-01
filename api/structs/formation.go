package structs

// ProcessFormation represents the formation for a Process
type ProcessFormation struct {
	Balancer string `json:"balancer"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Memory   int    `json:"memory"`
	CPU      int    `json:"cpu"`
	Ports    []int  `json:"ports"`
}

// Formation represents the formation for an App
type Formation []ProcessFormation
