package manifest

import (
	"fmt"
	"strings"
)

type Timer struct {
	Name string `yaml:"-"`

	Command  string `yaml:"command"`
	Schedule string `yaml:"schedule"`
	Service  string `yaml:"service"`
}

type Timers []Timer

func (t Timer) Cron() (string, error) {
	switch len(strings.Split(t.Schedule, " ")) {
	case 5:
		return fmt.Sprintf("%s *", t.Schedule), nil
	case 6:
		return t.Schedule, nil
	default:
		return "", fmt.Errorf("invalid schedule expression: %s", t.Schedule)
	}
}

func (t Timer) GetName() string {
	return t.Name
}

func (t *Timer) SetName(name string) error {
	t.Name = name
	return nil
}
