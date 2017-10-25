package manifest

type Resource struct {
	Name string `yaml:"-"`
	Type string `yaml:"type"`
}

type Resources []Resource

func (r Resource) GetName() string {
	return r.Name
}
