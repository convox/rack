package structs

type Service struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	StatusReason string `json:"status-reason"`
	Type         string `json:"type"`

	Apps    Apps              `json:"apps"`
	Exports map[string]string `json:"exports"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Services []Service
