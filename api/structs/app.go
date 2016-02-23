package structs

type App struct {
	Name    string `json:"name"`
	Release string `json:"release"`
	Status  string `json:"status"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Apps []App

func (s Apps) Len() int           { return len(s) }
func (s Apps) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s Apps) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
