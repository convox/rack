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

func TestCerts(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(structs.Certificates{*fxCertificate(), *fxCertificate()}, nil)

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID     DOMAIN       EXPIRES        ",
			"cert1  example.org  2 days from now",
			"cert1  example.org  2 days from now",
		})
	})
}

func TestCertsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestCertsDelete(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDelete", "cert1").Return(nil)

		res, err := testExecute(e, "certs delete cert1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Deleting certificate cert1... OK"})
	})
}

func TestCertsDeleteError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDelete", "cert1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "certs delete cert1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Deleting certificate cert1... "})
	})
}

func TestCertsGenerate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org", "other.example.org"}).Return(fxCertificate(), nil)

		res, err := testExecute(e, "certs generate test.example.org other.example.org", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Generating certificate... OK, cert1"})
	})
}

func TestCertsGenerateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org", "other.example.org"}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs generate test.example.org other.example.org", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Generating certificate... "})
	})
}

func TestCertsImport(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.CertificateCreateOptions{Chain: options.String("chain\n")}
		i.On("CertificateCreate", "cert\n", "key\n", opts).Return(fxCertificate(), nil)

		res, err := testExecute(e, "certs import testdata/cert.pem testdata/key.pem --chain testdata/chain.pem", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Importing certificate... OK, cert1"})
	})
}

func TestCertsImportError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.CertificateCreateOptions{Chain: options.String("chain\n")}
		i.On("CertificateCreate", "cert\n", "key\n", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs import testdata/cert.pem testdata/key.pem --chain testdata/chain.pem", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Importing certificate... "})
	})
}

func TestCertsImportClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.CertificateCreateOptions{Chain: options.String("chain\n")}
		i.On("CertificateCreateClassic", "cert\n", "key\n", opts).Return(fxCertificate(), nil)

		res, err := testExecute(e, "certs import testdata/cert.pem testdata/key.pem --chain testdata/chain.pem", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Importing certificate... OK, cert1"})
	})
}
