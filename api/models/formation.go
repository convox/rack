package models

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/api/provider"
)

type FormationEntry struct {
	Balancer string `json:"balancer"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Memory   int    `json:"memory"`
	CPU      int    `json:"cpu"`
	Ports    []int  `json:"ports"`
}

type Formation []FormationEntry

// FormationOptions carries the numeric dimensions that can change for a process type.
// Empty string indicates no change.
type FormationOptions struct {
	Count  string
	CPU    string
	Memory string
}

func ListFormation(app string) (Formation, error) {
	a, err := GetApp(app)
	if err != nil {
		return nil, err
	}

	if a.Release == "" {
		return Formation{}, nil
	}

	release, err := GetRelease(a.Name, a.Release)
	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(release.Manifest, a)
	if err != nil {
		return nil, err
	}

	formation := Formation{}

	for _, me := range manifest {
		var count, memory, cpu int

		if vals, ok := a.Parameters[fmt.Sprintf("%sFormation", UpperName(me.Name))]; ok {
			parts := strings.SplitN(vals, ",", 3)

			count, _ = strconv.Atoi(parts[0])
			cpu, _ = strconv.Atoi(parts[1])
			memory, _ = strconv.Atoi(parts[2])
		} else {
			count, _ = strconv.Atoi(a.Parameters[fmt.Sprintf("%sDesiredCount", UpperName(me.Name))])
			memory, _ = strconv.Atoi(a.Parameters[fmt.Sprintf("%sMemory", UpperName(me.Name))])
			cpu, _ = strconv.Atoi(a.Parameters[fmt.Sprintf("%sCpu", UpperName(me.Name))])
		}

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
			CPU:      cpu,
			Ports:    ports,
		})
	}

	sort.Sort(formation)

	return formation, nil
}

// Update Process Parameters for Count and Memory
// Expects -1 for memory and cpu and -2 for count to indicate no change, since count=0 is valid
func SetFormation(app, process string, opts FormationOptions) error {
	a, err := GetApp(app)
	if err != nil {
		return err
	}

	rel, err := GetRelease(a.Name, a.Release)
	if err != nil {
		return err
	}

	m, err := LoadManifest(rel.Manifest, a)
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

	if opts.Count != "" {
		_, err := strconv.Atoi(opts.Count)
		if err != nil {
			return err
		}
	}

	if opts.CPU != "" {
		cpu, err := strconv.Atoi(opts.CPU)
		if err != nil {
			return err
		}

		if int64(cpu) > capacity.InstanceCPU {
			return fmt.Errorf("requested cpu %d greater than instance size %d", cpu, capacity.InstanceCPU)
		}
	}

	if opts.Memory != "" {
		memory, err := strconv.Atoi(opts.Memory)
		if err != nil {
			return err
		}

		if int64(memory) > capacity.InstanceMemory {
			return fmt.Errorf("requested memory %d greater than instance size %d", memory, capacity.InstanceMemory)
		}
	}

	if vals, ok := a.Parameters[fmt.Sprintf("%sFormation", UpperName(process))]; ok {
		parts := strings.SplitN(vals, ",", 3)

		if opts.Count != "" {
			parts[0] = opts.Count
		}

		if opts.CPU != "" {
			parts[1] = opts.CPU
		}

		if opts.Memory != "" {
			parts[2] = opts.Memory
		}

		params[fmt.Sprintf("%sFormation", UpperName(process))] = strings.Join(parts, ",")
	} else {
		if opts.Count != "" {
			params[fmt.Sprintf("%sDesiredCount", UpperName(process))] = opts.Count
		}

		if opts.CPU != "" {
			params[fmt.Sprintf("%sCpu", UpperName(process))] = opts.CPU
		}

		if opts.Memory != "" {
			params[fmt.Sprintf("%sMemory", UpperName(process))] = opts.Memory
		}
	}

	NotifySuccess("release:scale", map[string]string{
		"app": rel.App,
		"id":  rel.Id,
	})

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
