package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

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

func TestServiceList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		s1 := structs.Services{fxService, fxService}
		s2 := structs.Services{}
		p.On("ServiceList", "app1").Return(s1, nil)
		err := c.Get("/apps/app1/services", stdsdk.RequestOptions{}, &s2)
		require.NoError(t, err)
		require.Equal(t, s1, s2)
	})
}

func TestServiceListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var s1 structs.Services
		p.On("ServiceList", "app1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/services", stdsdk.RequestOptions{}, &s1)
		require.EqualError(t, err, "err1")
		require.Nil(t, s1)
	})
}

func TestServiceUpdate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.ServiceUpdateOptions{
			Count:  options.Int(1),
			Cpu:    options.Int(2),
			Memory: options.Int(3),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"count":  "1",
				"cpu":    "2",
				"memory": "3",
			},
		}
		p.On("ServiceUpdate", "app1", "service1", opts).Return(nil)
		err := c.Put("/apps/app1/services/service1", ro, nil)
		require.NoError(t, err)
	})
}

func TestServiceUpdateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("ServiceUpdate", "app1", "service1", structs.ServiceUpdateOptions{}).Return(fmt.Errorf("err1"))
		err := c.Put("/apps/app1/services/service1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
