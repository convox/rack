package api_test

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/api"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	p := &structs.MockProvider{}
	p.On("Initialize", mock.Anything).Return(nil)

	s := api.NewWithProvider(p)
	s.Logger = logger.Discard
	s.Password = "pass1"
	s.Server.Recover = func(err error) {
		require.NoError(t, err, "httptest server panic")
	}

	ht := httptest.NewServer(s)
	defer ht.Close()

	c, err := sdk.New(ht.URL)
	require.NoError(t, err)

	res, err := c.GetStream("/apps", stdsdk.RequestOptions{})
	require.EqualError(t, err, "invalid authentication")
	require.Nil(t, res)

	u, err := url.Parse(ht.URL)
	require.NoError(t, err)

	u.User = url.UserPassword("convox", "pass1")

	c, err = sdk.New(u.String())
	require.NoError(t, err)

	_, err = c.GetStream("/auth", stdsdk.RequestOptions{})
	require.NoError(t, err)
}
