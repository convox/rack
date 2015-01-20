package provider

type Process struct {
	Name      string
	UpperName string
}

func ProcessList(cluster, app string) ([]Process, error) {
	return []Process{}, nil
}
