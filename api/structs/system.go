package structs

type System struct {
	Count   int    `json:"count"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	Version string `json:"version"`
}
