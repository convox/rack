package structs

type System struct {
	Count   int    `json:"count"`
	Name    string `json:"name"`
	Region  string `json:"region"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	Version string `json:"version"`
}
