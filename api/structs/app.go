package structs

import "os"

type App struct {
	Name    string `json:"name"`
	Release string `json:"release"`
	Status  string `json:"status"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Apps []App

// AppRepository defines an image repository for an App
type AppRepository struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

// IsBound checks if the app is bound returns true if it is, false otherwise
// If an app has a "Name" tag, it's considered bound
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

// StackName returns the app's stack if the app is bound. Otherwise returns the short name.
func (a *App) StackName() string {
	if a.IsBound() {
		return shortNameToStackName(a.Name)
	}

	return a.Name
}

func shortNameToStackName(appName string) string {
	rack := os.Getenv("RACK")

	if rack == appName {
		// Do no prefix the rack itself.
		return appName
	}

	return rack + "-" + appName
}
