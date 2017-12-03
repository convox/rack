package structs

type Service struct {
	Domain string        `json:"domain"`
	Ports  []ServicePort `json:"ports"`
	Name   string        `json:"name"`
}

type Services []Service

type ServicePort struct {
	Balancer    int    `json:"balancer"`
	Certificate string `json:"certificate"`
	Container   int    `json:"container"`
}

type ServiceUpdateOptions struct {
	Certificate string
}
