package structs

type App struct {
	Generation string `json:"generation,omitempty"`
	Name       string `json:"name"`
	Release    string `json:"release"`
	Sleep      bool   `json:"sleep"`
	Status     string `json:"status"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"parameters"`
	Tags       map[string]string `json:"-"`
}

type Apps []App

type AppCreateOptions struct {
	Generation *string `default:"2" flag:"generation,g" param:"generation"`
}

type AppUpdateOptions struct {
	Parameters map[string]string `param:"parameters"`
	Sleep      *bool             `param:"sleep"`
}

func (a Apps) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}
