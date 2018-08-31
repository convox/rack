package cli_test

import "github.com/convox/rack/pkg/structs"

var fxService = structs.Service{
	Name:   "service1",
	Count:  1,
	Cpu:    2,
	Domain: "domain",
	Memory: 3,
	Ports: []structs.ServicePort{
		{Balancer: 1, Certificate: "cert1", Container: 2},
		{Balancer: 1, Certificate: "cert1", Container: 2},
	},
}
