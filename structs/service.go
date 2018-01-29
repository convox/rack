package structs

type Service struct {
	Count  int           `json:"count"`
	Cpu    int           `json:"cpu"`
	Domain string        `json:"domain"`
	Memory int           `json:"memory"`
	Name   string        `json:"name"`
	Ports  []ServicePort `json:"ports"`
}

type Services []Service

type ServicePort struct {
	Balancer    int    `json:"balancer"`
	Certificate string `json:"certificate"`
	Container   int    `json:"container"`
}

type ServiceUpdateOptions struct {
	Count  *int
	Cpu    *int
	Memory *int
}
