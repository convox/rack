package helpers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
)

func useShared(t *testing.T) {
	creds, client, streaming := sessionCredentials, sessionHTTPClient, streamingHTTPClient
	age, timeout, header := credentialsMaxAge, requestTimeout, responseHeaderTimeout

	t.Cleanup(func() {
		sessionCredentials, sessionHTTPClient, streamingHTTPClient = creds, client, streaming
		credentialsMaxAge, requestTimeout, responseHeaderTimeout = age, timeout, header
	})
}

func credentialsJSON(expires time.Time) string {
	return fmt.Sprintf(`{"AccessKeyId":"a","SecretAccessKey":"b","Token":"c","Expiration":%q}`, expires.UTC().Format(time.RFC3339))
}

func credsServer(t *testing.T, creds func(n int32, w http.ResponseWriter)) (*httptest.Server, *int32) {
	var count int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/creds" {
			creds(atomic.AddInt32(&count, 1), w)
			return
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_, _ = fmt.Fprint(w, `{"clusters":[],"failures":[]}`)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", srv.URL+"/creds")

	return srv, &count
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func validFor(d time.Duration) func(int32, http.ResponseWriter) {
	return func(_ int32, w http.ResponseWriter) {
		_, _ = fmt.Fprint(w, credentialsJSON(time.Now().Add(d)))
	}
}

func describeClusters(t *testing.T, url string) error {
	s, err := NewSession()
	require.NoError(t, err)

	_, err = ecs.New(s, &aws.Config{Endpoint: aws.String(url), SleepDelay: func(time.Duration) {}}).DescribeClusters(&ecs.DescribeClustersInput{})

	return err
}

func TestSharedCredentialsOneFetch(t *testing.T) {
	isolate(t)
	useShared(t)

	srv, creds := credsServer(t, validFor(6*time.Hour))

	parallel := func() {
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				require.NoError(t, describeClusters(t, srv.URL))
			}()
		}
		wg.Wait()
	}

	parallel()
	require.EqualValues(t, 20, atomic.LoadInt32(creds))

	UseSharedCredentials()
	parallel()
	require.EqualValues(t, 21, atomic.LoadInt32(creds))
}

func TestSharedCredentialsExpiry(t *testing.T) {
	isolate(t)
	useShared(t)

	var valid atomic.Int64
	srv, _ := credsServer(t, func(_ int32, w http.ResponseWriter) {
		_, _ = fmt.Fprint(w, credentialsJSON(time.Now().Add(time.Duration(valid.Load()))))
	})

	require.False(t, CredentialsExpireWithin(6*time.Minute))

	UseSharedCredentials()
	require.False(t, CredentialsExpireWithin(6*time.Minute))

	valid.Store(int64(6 * time.Hour))
	require.NoError(t, describeClusters(t, srv.URL))
	require.False(t, CredentialsExpireWithin(6*time.Minute))

	sessionCredentials.Expire()
	valid.Store(int64(8 * time.Minute))
	require.NoError(t, describeClusters(t, srv.URL))
	require.True(t, CredentialsExpireWithin(6*time.Minute))
}

func TestSharedCredentialsMaxAge(t *testing.T) {
	isolate(t)
	useShared(t)

	srv, creds := credsServer(t, validFor(6*time.Hour))

	credentialsMaxAge = time.Second
	UseSharedCredentials()

	require.NoError(t, describeClusters(t, srv.URL))
	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 1, atomic.LoadInt32(creds))

	time.Sleep(1200 * time.Millisecond)

	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 2, atomic.LoadInt32(creds))
	require.False(t, CredentialsExpireWithin(6*time.Minute))
}

func TestSharedCredentialsErrorNotCached(t *testing.T) {
	isolate(t)
	useShared(t)

	srv, creds := credsServer(t, func(n int32, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `{"code":"InvalidRequest","message":"x"}`)
			return
		}
		_, _ = fmt.Fprint(w, credentialsJSON(time.Now().Add(6*time.Hour)))
	})

	UseSharedCredentials()

	require.Error(t, describeClusters(t, srv.URL))
	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 2, atomic.LoadInt32(creds))
}

func TestSharedCredentialsExpiredTokenNoPanic(t *testing.T) {
	isolate(t)
	useShared(t)

	srv, creds := credsServer(t, func(n int32, w http.ResponseWriter) {
		if n <= 2 {
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `{"code":"ExpiredToken","message":"x"}`)
			return
		}
		_, _ = fmt.Fprint(w, credentialsJSON(time.Now().Add(6*time.Hour)))
	})

	UseSharedCredentials()

	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 3, atomic.LoadInt32(creds))
}

func TestSharedCredentialsDisallowedHost(t *testing.T) {
	isolate(t)
	useShared(t)

	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "http://192.0.2.1/creds")

	UseSharedCredentials()
	require.Nil(t, sessionCredentials)
}

func TestSharedTimeoutsHeaderStall(t *testing.T) {
	isolate(t)
	useShared(t)

	buf := &bytes.Buffer{}
	useRequestPath(t, buf)

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			select {
			case <-time.After(5 * time.Second):
			case <-r.Context().Done():
			}
			return
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_, _ = fmt.Fprint(w, `{"clusters":[],"failures":[]}`)
	}))
	t.Cleanup(srv.Close)

	requestTimeout, responseHeaderTimeout = 10*time.Second, time.Second
	UseSharedCredentials()

	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 2, atomic.LoadInt32(&calls))
	require.Equal(t, 1, strings.Count(buf.String(), "\n"))
	require.Contains(t, buf.String(), "code=RequestError status=0")
}

func TestSharedTimeoutsBodyStall(t *testing.T) {
	isolate(t)
	useShared(t)

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		if atomic.AddInt32(&calls, 1) == 1 {
			_, _ = fmt.Fprint(w, `{"clusters":[`)
			flush(w)
			select {
			case <-time.After(5 * time.Second):
			case <-r.Context().Done():
			}
			return
		}
		_, _ = fmt.Fprint(w, `{"clusters":[],"failures":[]}`)
	}))
	t.Cleanup(srv.Close)

	requestTimeout, responseHeaderTimeout = 2*time.Second, time.Second
	UseSharedCredentials()

	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 2, atomic.LoadInt32(&calls))
}

func TestSharedTimeoutsMonitor(t *testing.T) {
	isolate(t)
	useShared(t)

	srv, creds := credsServer(t, func(n int32, w http.ResponseWriter) {
		if n == 1 {
			time.Sleep(3 * time.Second)
		}
		_, _ = fmt.Fprint(w, credentialsJSON(time.Now().Add(6*time.Hour)))
	})

	requestTimeout, responseHeaderTimeout = 2*time.Second, time.Second
	UseSharedCredentials()

	require.NoError(t, describeClusters(t, srv.URL))
	require.EqualValues(t, 2, atomic.LoadInt32(creds))

	s, err := NewSession()
	require.NoError(t, err)
	require.Same(t, sessionHTTPClient, s.Config.HTTPClient)
}

func TestStreamingClientNoTimeout(t *testing.T) {
	isolate(t)
	useShared(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "6")
		w.WriteHeader(200)
		for i := 0; i < 6; i++ {
			_, _ = w.Write([]byte("x"))
			flush(w)
			select {
			case <-time.After(500 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	requestTimeout, responseHeaderTimeout = 2*time.Second, time.Second
	UseSharedCredentials()

	read := func(c *http.Client) error {
		s, err := NewSession()
		require.NoError(t, err)

		cfg := &aws.Config{Endpoint: aws.String(srv.URL), S3ForcePathStyle: aws.Bool(true)}
		if c != nil {
			cfg.HTTPClient = c
		}

		res, err := s3.New(s, cfg).GetObject(&s3.GetObjectInput{Bucket: aws.String("b"), Key: aws.String("k")})
		require.NoError(t, err)
		defer func() { _ = res.Body.Close() }()

		_, err = io.ReadAll(res.Body)
		return err
	}

	require.Error(t, read(nil))
	require.NoError(t, read(StreamingHTTPClient()))
}

func TestSharedCredentialsStaticKeys(t *testing.T) {
	isolate(t)
	useShared(t)

	credsServer(t, validFor(6*time.Hour))

	UseSharedCredentials()
	require.NotNil(t, sessionCredentials)

	t.Setenv("AWS_ACCESS_KEY_ID", "test")

	UseSharedCredentials()
	require.Nil(t, sessionCredentials)
	require.NotNil(t, sessionHTTPClient)
}
