package main

type Process struct {
	Balancer bool
	Name     string
}

func buildProcesses(processes, balancers []string) []Process {
	procs := make([]Process, len(processes))

	for i, p := range processes {
		balanced := false

		for _, b := range balancers {
			if b == p {
				balanced = true
				break
			}
		}

		procs[i] = Process{
			Name:     p,
			Balancer: balanced,
		}
	}

	return procs
}
