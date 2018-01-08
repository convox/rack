package structs

type App struct {
	Generation string `json:"generation,omitempty"`
	Name       string `json:"name"`
	Release    string `json:"release"`
	Status     string `json:"status"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Apps []App

type AppCreateOptions struct {
	Generation *string
}

type AppUpdateOptions struct {
	Parameters map[string]string
}

func (a Apps) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}
