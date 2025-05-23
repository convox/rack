package manifest

type Resource struct {
	Name    string            `yaml:"-"`
	Type    string            `yaml:"type"`
	Options map[string]string `yaml:"options"`
	Tags    map[string]string `yaml:"tags"`
}

type Resources []Resource

func (r Resource) GetName() string {
	return r.Name
}
