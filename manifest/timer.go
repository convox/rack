package manifest

type Timer struct {
	Name string `yaml:"-"`

	Command  string `yaml:"command"`
	Schedule string `yaml:"schedule"`
	Service  string `yaml:"service"`
}

type Timers []Timer

func (t Timer) GetName() string {
	return t.Name
}

func (t *Timer) SetName(name string) error {
	t.Name = name
	return nil
}
