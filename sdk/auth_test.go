package sdk_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/rack/sdk"
	"github.com/convox/stdapi"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	s := stdapi.New("api", "api")
	s.Route("GET", "/auth", func(c *stdapi.Context) error {
		return c.RenderJSON(struct {
			Id string
		}{"1234"})
	})
	testServer(t, s, func(c *sdk.Client) {
		id, err := c.Auth()
		require.NoError(t, err)
		require.Equal(t, "1234", id)
	})
}

func testServer(t *testing.T, handler http.Handler, fn func(*sdk.Client)) {
	ht := httptest.NewServer(handler)
	defer ht.Close()

	c, err := sdk.New(ht.URL)
	require.NoError(t, err)

	fn(c)
}
