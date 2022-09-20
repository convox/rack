package helpers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestEndpointCheck(t *testing.T) {
	ht := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		return
	}))
	defer ht.Close()

	err := helpers.EndpointCheck(ht.URL)
	require.NoError(t, err)
}

func TestEndpointWait(t *testing.T) {
	ht := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		return
	}))
	defer ht.Close()

	err := helpers.EndpointWait(ht.URL)
	require.NoError(t, err)
}
