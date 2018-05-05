package api_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/convox/logger"
	"github.com/convox/rack/api"
	"github.com/convox/rack/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, fn func(*stdsdk.Client, *structs.MockProvider)) {
	s := api.New()
	s.Logger = logger.Discard
	s.Server.Recover = func(err error) {
		require.NoError(t, err, "httptest server panic")
	}

	p := &structs.MockProvider{}
	p.On("WithContext", mock.MatchedBy(requestContextMatcher)).Return(p)
	s.Provider = p

	ht := httptest.NewServer(s)
	defer ht.Close()

	c, err := stdsdk.New(ht.URL)
	require.NoError(t, err)

	fn(c, p)
}

func requestContextMatcher(ctx context.Context) bool {
	_, ok := ctx.Value("request.id").(string)
	return ok
}
