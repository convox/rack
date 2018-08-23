package structs

// ProcessFormation represents the formation for a Process
type ProcessFormation struct {
	Count    int    `json:"count"`
	CPU      int    `json:"cpu"`
	Hostname string `json:"hostname"`
	Name     string `json:"name"`
	Memory   int    `json:"memory"`
	Ports    []int  `json:"ports"`
}

// Formation represents the formation for an App
type Formation []ProcessFormation
