package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestServices(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS",
			"service1  domain  1:2 1:2",
			"service1  domain  1:2 1:2",
		})
	})
}

func TestServicesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestServicesClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("FormationGet", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS",
			"service1  domain  1:2 1:2",
			"service1  domain  1:2 1:2",
		})
	})
}

func TestServicesRestart(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ServiceRestart", "app1", "service1").Return(nil)

		res, err := testExecute(e, "services restart service1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Restarting service1... OK"})
	})
}

func TestServicesRestartError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ServiceRestart", "app1", "service1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "services restart service1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Restarting service1... "})
	})
}

func TestServicesWithNLB(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(structs.Services{*fxServiceNLB(), *fxService()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443 9443:8080(internal)",
			"service1  domain  1:2 1:2  ",
		})
	})
}

func TestServicesAllWithNLB(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(structs.Services{*fxServiceNLB(), *fxServiceNLB()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443 9443:8080(internal)",
			"service1  domain  1:2 1:2  8443:8443 9443:8080(internal)",
		})
	})
}

func TestServicesWithNLBTLS(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(structs.Services{*fxServiceNLBTLS(), *fxService()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8080/tls 9443:8080(internal)",
			"service1  domain  1:2 1:2  ",
		})
	})
}

func TestServicesMixedSchemes(t *testing.T) {
	t.Run("public first preserves order", func(t *testing.T) {
		testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
			i.On("SystemGet").Return(fxSystem(), nil)
			s := fxService()
			s.Nlb = []structs.ServiceNlbPort{
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
			}
			i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

			res, err := testExecute(e, "services -a app1", nil)
			require.NoError(t, err)
			require.Equal(t, 0, res.Code)
			res.RequireStdout(t, []string{
				"SERVICE   DOMAIN  PORTS    NLB PORTS",
				"service1  domain  1:2 1:2  8443:8443 9443:8080(internal)",
			})
		})
	})

	t.Run("internal first preserves order", func(t *testing.T) {
		testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
			i.On("SystemGet").Return(fxSystem(), nil)
			s := fxService()
			s.Nlb = []structs.ServiceNlbPort{
				{Port: 9443, Protocol: "tcp", ContainerPort: 8080, Scheme: "internal"},
				{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
			}
			i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

			res, err := testExecute(e, "services -a app1", nil)
			require.NoError(t, err)
			require.Equal(t, 0, res.Code)
			res.RequireStdout(t, []string{
				"SERVICE   DOMAIN  PORTS    NLB PORTS",
				"service1  domain  1:2 1:2  9443:8080(internal) 8443:8443",
			})
		})
	})
}

func TestServicesEmptyNlbSlice(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		s := fxService()
		s.Nlb = []structs.ServiceNlbPort{}
		i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS",
			"service1  domain  1:2 1:2",
		})
	})
}

func TestServicesWorkerOnlyNLB(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		worker := &structs.Service{
			Name: "worker",
			Nlb:  []structs.ServiceNlbPort{{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"}},
		}
		web := &structs.Service{
			Name:   "web",
			Domain: "domain",
			Ports:  []structs.ServicePort{{Balancer: 1, Container: 2}},
		}
		i.On("ServiceList", "app1").Return(structs.Services{*worker, *web}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE  DOMAIN  PORTS  NLB PORTS",
			"worker                  8443:8443",
			"web      domain  1:2    ",
		})
	})
}

func TestServicesNLBAllDefaults(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		s := fxService()
		s.Nlb = []structs.ServiceNlbPort{
			{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public"},
		}
		i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443",
		})
	})
}

func TestServicesNLBCrossZoneOnly(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		s := fxService()
		s.Nlb = []structs.ServiceNlbPort{
			{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public", CrossZone: options.Bool(true)},
		}
		i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443[cz=true]",
		})
	})
}

func TestServicesNLBCrossZoneFalse(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		s := fxService()
		s.Nlb = []structs.ServiceNlbPort{
			{Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public", CrossZone: options.Bool(false)},
		}
		i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443[cz=false]",
		})
	})
}

func TestServicesNLBAllowCIDR(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		s := fxService()
		s.Nlb = []structs.ServiceNlbPort{
			{
				Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public",
				AllowCIDR: []string{"10.0.0.0/24", "10.1.0.0/24"},
			},
		}
		i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443[allow=2]",
		})
	})
}

func TestServicesNLBAllThree(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		s := fxService()
		s.Nlb = []structs.ServiceNlbPort{
			{
				Port: 8443, Protocol: "tcp", ContainerPort: 8443, Scheme: "public",
				CrossZone:        options.Bool(true),
				AllowCIDR:        []string{"10.0.0.0/24", "10.1.0.0/24"},
				PreserveClientIP: options.Bool(false),
			},
		}
		i.On("ServiceList", "app1").Return(structs.Services{*s}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS    NLB PORTS",
			"service1  domain  1:2 1:2  8443:8443[cz=true allow=2 pcip=false]",
		})
	})
}
