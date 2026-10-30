package api_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
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

func TestNewSharesCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROVIDER", "test")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	var creds int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/creds" {
			if atomic.AddInt32(&creds, 1) == 1 {
				w.WriteHeader(429)
				_, _ = fmt.Fprint(w, "You have reached maximum request limit.")
				return
			}
			_, _ = fmt.Fprintf(w, `{"AccessKeyId":"a","SecretAccessKey":"b","Token":"c","Expiration":%q}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
			return
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_, _ = fmt.Fprint(w, `{"clusters":[],"failures":[]}`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", srv.URL+"/creds")

	t.Cleanup(func() {
		require.NoError(t, os.Setenv("AWS_ACCESS_KEY_ID", "test"))
		helpers.UseSharedCredentials()
	})

	buf := &bytes.Buffer{}
	old := logger.Output
	logger.Output = buf
	t.Cleanup(func() { logger.Output = old })

	provider.Mock.On("Initialize", mock.Anything).Return(nil)
	provider.Mock.On("SystemJwtSignKey").Return("", nil)

	_, err := api.New()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		s, err := helpers.NewSession()
		require.NoError(t, err)

		_, err = ecs.New(s, &aws.Config{Endpoint: aws.String(srv.URL)}).DescribeClusters(&ecs.DescribeClustersInput{})
		require.NoError(t, err)
	}

	require.EqualValues(t, 2, atomic.LoadInt32(&creds))
	require.Contains(t, buf.String(), "service=CredentialsEndpoint operation=GetCredentials code=SerializationError status=429")
}
