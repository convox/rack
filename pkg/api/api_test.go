package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/convox/logger"
	"github.com/convox/rack/pkg/api"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/stdapi"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, fn func(*stdsdk.Client, *structs.MockProvider)) {
	p := &structs.MockProvider{}
	p.On("Initialize", mock.Anything).Return(nil)
	p.On("WithContext", mock.Anything).Return(p).Maybe()
	p.On("SystemJwtSignKey").Return("", nil)

	s := api.NewWithProvider(p)
	s.Logger = logger.Discard
	s.Server.Recover = func(err error, c *stdapi.Context) {
		require.NoError(t, err, "httptest server panic")
	}

	ht := httptest.NewServer(s)
	defer ht.Close()

	c, err := stdsdk.New(ht.URL)
	require.NoError(t, err)

	fn(c, p)

	p.AssertExpectations(t)
}

func requestContextMatcher(ctx context.Context) bool {
	_, ok := ctx.Value("request.id").(string)
	return ok
}

func TestNewUsesRequestPathRetries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROVIDER", "test")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")

	old := logger.Output
	logger.Output = io.Discard
	t.Cleanup(func() { logger.Output = old })

	provider.Mock.On("Initialize", mock.Anything).Return(nil)
	provider.Mock.On("SystemJwtSignKey").Return("", nil)

	_, err := api.New()
	require.NoError(t, err)

	s, err := helpers.NewSession()
	require.NoError(t, err)

	r, ok := s.Config.Retryer.(request.Retryer)
	require.True(t, ok)

	d := r.RetryRules(&request.Request{Error: awserr.New("ThrottlingException", "x", nil), HTTPResponse: &http.Response{StatusCode: 400}})
	require.Less(t, d, time.Second)
}

func TestCheck(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		res, err := c.GetStream("/check", stdsdk.RequestOptions{})
		require.NoError(t, err)
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "ok", string(data))
	})
}
