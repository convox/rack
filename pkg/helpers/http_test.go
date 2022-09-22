package helpers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	testText := "hello world"
	ht := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testText))
		return
	}))
	defer ht.Close()

	resp, err := helpers.Get(ht.URL)
	require.NoError(t, err)
	require.Equal(t, testText, string(resp))
}
