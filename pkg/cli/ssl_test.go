package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestSsl(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)
		i.On("CertificateList").Return(structs.Certificates{*fxCertificate()}, nil)

		res, err := testExecute(e, "ssl -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ENDPOINT    CERTIFICATE  DOMAIN       EXPIRES        ",
			"service1:1  cert1        example.org  2 days from now",
			"service1:1  cert1        example.org  2 days from now",
			"service1:1  cert1        example.org  2 days from now",
			"service1:1  cert1        example.org  2 days from now",
		})
	})
}

func TestSslError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "ssl -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestSslClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("FormationGet", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)
		i.On("CertificateList").Return(structs.Certificates{*fxCertificate()}, nil)

		res, err := testExecute(e, "ssl -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ENDPOINT    CERTIFICATE  DOMAIN       EXPIRES        ",
			"service1:1  cert1        example.org  2 days from now",
			"service1:1  cert1        example.org  2 days from now",
			"service1:1  cert1        example.org  2 days from now",
			"service1:1  cert1        example.org  2 days from now",
		})
	})
}

func TestSslUpdate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(fxApp(), nil)

		res, err := testExecute(e, "ssl update web:5000 cert1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: command not valid for generation 2 applications"})
		res.RequireStdout(t, []string{""})
	})
}

func TestSslUpdateGeneration1(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(fxAppGeneration1(), nil)
		i.On("CertificateApply", "app1", "web", 5000, "cert1").Return(nil)

		res, err := testExecute(e, "ssl update web:5000 cert1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating certificate... OK"})
	})
}

func TestSslUpdateGeneration1Error(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(fxAppGeneration1(), nil)
		i.On("CertificateApply", "app1", "web", 5000, "cert1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "ssl update web:5000 cert1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Updating certificate... "})
	})
}
