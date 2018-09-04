package api_test

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/api"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, fn func(*stdsdk.Client, *structs.MockProvider)) {
	p := &structs.MockProvider{}
	p.On("Initialize", mock.Anything).Return(nil)

	s := api.NewWithProvider(p)
	s.Logger = logger.Discard
	s.Server.Recover = func(err error) {
		require.NoError(t, err, "httptest server panic")
	}

	ht := httptest.NewServer(s)
	defer ht.Close()

	c, err := stdsdk.New(ht.URL)
	require.NoError(t, err)

	fn(c, p)

	// p.AssertExpectations(t)
}

func requestContextMatcher(ctx context.Context) bool {
	_, ok := ctx.Value("request.id").(string)
	return ok
}

func TestCheck(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		res, err := c.GetStream("/check", stdsdk.RequestOptions{})
		require.NoError(t, err)
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "ok\n", string(data))
	})
}
