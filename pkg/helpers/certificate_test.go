package helpers_test

import (
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestCertificate(t *testing.T) {
	cert, err := helpers.CertificateSelfSigned("convox.com")
	require.NoError(t, err)

	_, err = helpers.CertificateCA("convox.com", cert)
	require.NoError(t, err)
}
