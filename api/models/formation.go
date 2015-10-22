package models

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

type FormationEntry struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Memory int    `json:"memory"`
	Ports  []int  `json:"ports"`
}

type Formation []FormationEntry

func ListFormation(app string) (Formation, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	release, err := a.LatestRelease()

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
			Name:   me.Name,
			Count:  count,
			Memory: memory,
			Ports:  ports,
		})
	}

	sort.Sort(formation)

	return formation, nil
}

func SetFormation(app, process, count, memory string) error {
	a, err := GetApp(app)

	if err != nil {
		return err
	}

	rel, err := a.LatestRelease()

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

	system, err := GetSystem()

	if err != nil {
		return err
	}

	params := map[string]string{}

	if count != "" {
		c, err := strconv.Atoi(count)

		if err != nil {
			return err
		}

		// if the app has external ports we can only have n-1 instances of it
		// because elbs expect the process to be available at the same port on
		// every instance and we need room for the rolling updates
		if len(me.ExternalPorts()) > 0 && c >= system.Count {
			return fmt.Errorf("rack has %d instances, can't scale processes beyond %d", system.Count, system.Count-1)
		}

		params[fmt.Sprintf("%sDesiredCount", UpperName(process))] = count
	}

	if memory != "" {
		params[fmt.Sprintf("%sMemory", UpperName(process))] = memory
	}

	return a.UpdateParams(params)
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
