package models

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/convox/rack/api/provider"
)

type FormationEntry struct {
	Balancer string `json:"balancer"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Memory   int    `json:"memory"`
	Ports    []int  `json:"ports"`
}

type Formation []FormationEntry

func ListFormation(app string) (Formation, error) {
	a, err := provider.AppGet(app)

	if err != nil {
		return nil, err
	}

	release, err := appLatestRelease(a)

	if err != nil {
		return nil, err
	}

	if release == nil {
		return Formation{}, nil
	}

	manifest, err := LoadManifest(release.Manifest)

	if err != nil {
		return nil, err
	}

	formation := Formation{}

	for _, me := range manifest {
		count, _ := strconv.Atoi(a.Parameters[fmt.Sprintf("%sDesiredCount", UpperName(me.Name))])
		memory, _ := strconv.Atoi(a.Parameters[fmt.Sprintf("%sMemory", UpperName(me.Name))])

		re := regexp.MustCompile(fmt.Sprintf(`%sPort(\d+)Host`, UpperName(me.Name)))

		ports := []int{}

		for key, _ := range a.Parameters {
			matches := re.FindStringSubmatch(key)

			if len(matches) == 2 {
				port, _ := strconv.Atoi(matches[1])
				ports = append(ports, port)
			}
		}

		formation = append(formation, FormationEntry{
			Balancer: first(a.Outputs[fmt.Sprintf("Balancer%sHost", UpperName(me.Name))], a.Outputs["BalancerHost"]),
			Name:     me.Name,
			Count:    count,
			Memory:   memory,
			Ports:    ports,
		})
	}

	sort.Sort(formation)

	return formation, nil
}

func SetFormation(app, process string, count, memory int64) error {
	a, err := provider.AppGet(app)

	if err != nil {
		return err
	}

	rel, err := appLatestRelease(a)

	if err != nil {
		return err
	}

	m, err := LoadManifest(rel.Manifest)

	if err != nil {
		return err
	}

	me := m.Entry(process)

	if me == nil {
		return fmt.Errorf("no such process: %s", process)
	}

	capacity, err := provider.CapacityGet()

	if err != nil {
		return err
	}

	params := map[string]string{}

	if count > 0 {
		params[fmt.Sprintf("%sDesiredCount", UpperName(process))] = fmt.Sprintf("%d", count)
	}

	if memory > 0 {
		if memory > capacity.InstanceMemory {
			return fmt.Errorf("requested memory %d greater than instance size %d", memory, capacity.InstanceMemory)
		}

		params[fmt.Sprintf("%sMemory", UpperName(process))] = fmt.Sprintf("%d", memory)
	}

	provider.NotifySuccess("release:scale", map[string]string{
		"app": rel.App,
		"id":  rel.Id,
	})

	return AppUpdateParams(a, params)
}

func (f Formation) Entry(name string) *FormationEntry {
	for _, fe := range f {
		if fe.Name == name {
			return &fe
		}
	}

	return nil
}

func (f Formation) Len() int {
	return len(f)
}

func (f Formation) Less(a, b int) bool {
	return f[a].Name < f[b].Name
}

func (f Formation) Swap(a, b int) {
	f[a], f[b] = f[b], f[a]
}
