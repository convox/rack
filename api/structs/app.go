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

func (a *App) IsBound() bool {
	if a.Tags == nil {
		// Default to bound.
		return true
	}

	if _, ok := a.Tags["Name"]; ok {
		// Bound apps MUST have a "Name" tag.
		return true
	}

	// Tags are present but "Name" tag is not, so we have an unbound app.
	return false
}
