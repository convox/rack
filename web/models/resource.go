package models

import "fmt"

type Resource struct {
	Name string
	Type string

	App string
}

type Resources []Resource

func (r *Resource) Env() (string, error) {
	env, err := buildTemplate(r.Type, "env", r)

	if err != nil {
		return "", err
	}

	return env, nil
}

func (r *Resource) Formation() (string, error) {
	formation, err := buildTemplate(r.Type, "formation", r)

	if err != nil {
		return "", err
	}

	return formation, nil
}

func (r Resource) AvailabilityZones() []string {
	azs := []string{}

	for _, subnet := range ListSubnets() {
		azs = append(azs, subnet.AvailabilityZone)
	}

	return azs
}

func (r Resource) FormationName() string {
	return fmt.Sprintf("%s%s", upperName(r.Type), upperName(r.Name))
}
