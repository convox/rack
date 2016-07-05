package models

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/manifest"
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

	manifest, err := manifest.Load([]byte(release.Manifest))
	if err != nil {
		return nil, err
	}

	formation := Formation{}

	for _, me := range manifest.Services {
		var count, memory, cpu int

		if vals, ok := a.Parameters[fmt.Sprintf("%sFormation", UpperName(me.Name))]; ok {
			parts := strings.SplitN(vals, ",", 3)
			if len(parts) != 3 {
				return nil, fmt.Errorf("%s formation settings not in Count,Cpu,Memory format", me.Name)
			}

			count, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("%s %s not numeric", me.Name, "count")
			}

			cpu, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("%s %s not numeric", me.Name, "CPU")
			}

			memory, err = strconv.Atoi(parts[2])
			if err != nil {
				return nil, fmt.Errorf("%s %s not numeric", me.Name, "memory")
			}
		} else {
			count, err = strconv.Atoi(a.Parameters[fmt.Sprintf("%sDesiredCount", UpperName(me.Name))])
			if err != nil {
				return nil, fmt.Errorf("%s %s not numeric", me.Name, "count")
			}

			// backwards compatibility: old stacks that do not have a WebCpu Parameter should return 0, not an error
			if c, ok := a.Parameters[fmt.Sprintf("%sCpu", UpperName(me.Name))]; ok {
				cpu, err = strconv.Atoi(c)
				if err != nil {
					return nil, fmt.Errorf("%s %s not numeric", me.Name, "cpu")
				}
			}

			memory, err = strconv.Atoi(a.Parameters[fmt.Sprintf("%sMemory", UpperName(me.Name))])
			if err != nil {
				return nil, fmt.Errorf("%s %s not numeric", me.Name, "memory")
			}
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
// Empty string for opts.Count, opts.CPU or opts.Memory indicates no change, since count=0 is valid
func SetFormation(app, process string, opts FormationOptions) error {
	a, err := GetApp(app)
	if err != nil {
		return err
	}

	rel, err := GetRelease(a.Name, a.Release)
	if err != nil {
		return err
	}

	m, err := manifest.Load([]byte(rel.Manifest))
	if err != nil {
		return err
	}

	_, ok := m.Services[process]
	if !ok {
		return fmt.Errorf("no such process: %s", process)
	}

	capacity, err := provider.CapacityGet()
	if err != nil {
		return err
	}

	params := map[string]string{}

	if opts.Count != "" {
		count, err := strconv.Atoi(opts.Count)
		if err != nil {
			return err
		}

		if count < -1 {
			return fmt.Errorf("requested count %d must -1 or greater", count)
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

		if cpu < 0 {
			return fmt.Errorf("requested cpu %d must be zero or greater", cpu)
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

		if memory < 1 {
			return fmt.Errorf("requested memory %d must be greater than zero", memory)
		}
	}

	if vals, ok := a.Parameters[fmt.Sprintf("%sFormation", UpperName(process))]; ok {
		parts := strings.SplitN(vals, ",", 3)
		if len(parts) != 3 {
			return fmt.Errorf("%s formation settings not in Count,Cpu,Memory format", process)
		}

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
