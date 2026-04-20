package structs

type Service struct {
	Count  int              `json:"count"`
	Cpu    int              `json:"cpu"`
	Domain string           `json:"domain"`
	Memory int              `json:"memory"`
	Name   string           `json:"name"`
	Nlb    []ServiceNlbPort `json:"nlb"`
	Ports  []ServicePort    `json:"ports"`
}

type Services []Service

// ServiceNlbPort corresponds to manifest.ServiceNLBPort. Naming diverges
// intentionally to match the pkg/structs casing convention (Cpu, Nlb) rather
// than the manifest package's all-caps initialism style.
type ServiceNlbPort struct {
	ContainerPort int    `json:"container-port"`
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"`
	Scheme        string `json:"scheme"`
	Certificate   string `json:"certificate"`
}

type ServicePort struct {
	Balancer    int    `json:"balancer"`
	Certificate string `json:"certificate"`
	Container   int    `json:"container"`
}

type ServiceUpdateOptions struct {
	Count  *int `flag:"count" param:"count"`
	Cpu    *int `flag:"cpu" param:"cpu"`
	Memory *int `flag:"memory" param:"memory"`
}
