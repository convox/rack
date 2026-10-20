package helpers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/logger"
	"github.com/stretchr/testify/require"
)

func isolate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_ACCESS_KEY", "")
	t.Setenv("AWS_SECRET_KEY", "")
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")
	t.Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", "")
}

func useRequestPath(t *testing.T, w io.Writer) {
	old := sessionRetryer
	t.Cleanup(func() { sessionRetryer = old })

	UseRequestPathRetries()

	r, ok := sessionRetryer.(loggedRetryer)
	require.True(t, ok)

	r.log = logger.NewWriter("ns=aws", w).At("retry")
	sessionRetryer = r
}

func throttleServer(t *testing.T, n int32) (*httptest.Server, *int32) {
	var count int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&count, 1)
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		if n < 0 || c <= n {
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `{"__type":"ThrottlingException","message":"Rate exceeded"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"clusters":[],"failures":[]}`)
	}))
	t.Cleanup(srv.Close)

	return srv, &count
}

func ecsClient(t *testing.T, url string, delays *[]time.Duration) *ecs.ECS {
	s, err := NewSession()
	require.NoError(t, err)

	return ecs.New(s, &aws.Config{
		Endpoint:   aws.String(url),
		SleepDelay: func(d time.Duration) { *delays = append(*delays, d) },
	})
}

func TestRetryProfileDelays(t *testing.T) {
	def := sessionRetryer
	useRequestPath(t, io.Discard)
	rp := sessionRetryer

	throttle := func(n int) *request.Request {
		return &request.Request{RetryCount: n, Error: awserr.New("ThrottlingException", "x", nil), HTTPResponse: &http.Response{StatusCode: 400}}
	}
	other := func(n int) *request.Request {
		return &request.Request{RetryCount: n, Error: awserr.New("InternalError", "x", nil), HTTPResponse: &http.Response{StatusCode: 500}}
	}

	type band struct{ lo, hi time.Duration }
	s := time.Second

	tests := []struct {
		name    string
		retryer request.Retryer
		req     func(int) *request.Request
		bands   []band
	}{
		{"default throttle", def, throttle, []band{{10 * s, 20 * s}, {20 * s, 40 * s}, {30 * s, 60 * s}}},
		{"request-path throttle", rp, throttle, []band{{s / 2, s}, {s, 2 * s}, {2 * s, 4 * s}, {s * 5 / 2, 5 * s}}},
		{"default other", def, other, []band{{s, 2 * s}, {2 * s, 4 * s}, {s * 5 / 2, 5 * s}}},
		{"request-path other", rp, other, []band{{s, 2 * s}, {2 * s, 4 * s}, {s * 5 / 2, 5 * s}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, MAX_RETRY, tt.retryer.MaxRetries())

			for n := 0; n < MAX_RETRY; n++ {
				b := tt.bands[len(tt.bands)-1]
				if n < len(tt.bands) {
					b = tt.bands[n]
				}

				for i := 0; i < 200; i++ {
					d := tt.retryer.RetryRules(tt.req(n))
					require.True(t, d >= b.lo && d <= b.hi, "retry %d: %s not in [%s, %s]", n, d, b.lo, b.hi)
				}
			}
		})
	}
}

func TestNewSessionThrottleThenAnswer(t *testing.T) {
	isolate(t)

	tests := []struct {
		name        string
		requestPath bool
		lo, hi      time.Duration
	}{
		{"default", false, 10 * time.Second, 20 * time.Second},
		{"request-path", true, 500 * time.Millisecond, time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.requestPath {
				useRequestPath(t, io.Discard)
			}

			srv, count := throttleServer(t, 2)

			var delays []time.Duration
			_, err := ecsClient(t, srv.URL, &delays).DescribeClusters(&ecs.DescribeClustersInput{})
			require.NoError(t, err)
			require.EqualValues(t, 3, atomic.LoadInt32(count))
			require.Len(t, delays, 2)
			require.True(t, delays[0] >= tt.lo && delays[0] < tt.hi, "first wait %s", delays[0])
			require.True(t, delays[1] >= 2*tt.lo && delays[1] < 2*tt.hi, "second wait %s", delays[1])
		})
	}
}

func TestRetryLogLines(t *testing.T) {
	isolate(t)
	buf := &bytes.Buffer{}
	useRequestPath(t, buf)
	srv, count := throttleServer(t, 2)

	var delays []time.Duration
	_, err := ecsClient(t, srv.URL, &delays).DescribeClusters(&ecs.DescribeClustersInput{})
	require.NoError(t, err)
	require.EqualValues(t, 3, atomic.LoadInt32(count))
	require.Len(t, delays, 2)

	expected := ""
	for i, d := range delays {
		expected += fmt.Sprintf("ns=aws at=retry service=ecs operation=DescribeClusters code=ThrottlingException status=400 attempt=%d delay=%d\n", i+1, d.Milliseconds())
	}
	require.Equal(t, expected, buf.String())
}

func TestRetryLogNoLineWithoutRetry(t *testing.T) {
	isolate(t)
	buf := &bytes.Buffer{}
	useRequestPath(t, buf)

	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-amz-json") {
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `{"__type":"ClientException","message":"nope"}`)
			return
		}
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(400)
		_, _ = fmt.Fprint(w, `<ErrorResponse><Error><Type>Sender</Type><Code>ValidationError</Code><Message>Stack with id x does not exist</Message></Error><RequestId>1</RequestId></ErrorResponse>`)
	}))
	t.Cleanup(srv.Close)

	var delays []time.Duration
	_, err := ecsClient(t, srv.URL, &delays).DescribeClusters(&ecs.DescribeClustersInput{})
	require.Error(t, err)
	require.EqualValues(t, 1, atomic.LoadInt32(&count))

	s, err := NewSession()
	require.NoError(t, err)
	cf := cloudformation.New(s, &aws.Config{
		Endpoint:   aws.String(srv.URL),
		SleepDelay: func(d time.Duration) { delays = append(delays, d) },
	})
	_, err = cf.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String("x")})
	require.Error(t, err)
	aerr, ok := err.(awserr.Error)
	require.True(t, ok)
	require.Equal(t, "ValidationError", aerr.Code())
	require.EqualValues(t, 2, atomic.LoadInt32(&count))

	require.Empty(t, delays)
	require.Empty(t, buf.String())
}

func TestRetryLogExhausted(t *testing.T) {
	isolate(t)
	buf := &bytes.Buffer{}
	useRequestPath(t, buf)
	srv, count := throttleServer(t, -1)

	var delays []time.Duration
	_, err := ecsClient(t, srv.URL, &delays).DescribeClusters(&ecs.DescribeClustersInput{})
	require.Error(t, err)
	aerr, ok := err.(awserr.Error)
	require.True(t, ok)
	require.Equal(t, "ThrottlingException", aerr.Code())
	require.EqualValues(t, 11, atomic.LoadInt32(count))
	require.Len(t, delays, 10)

	var sum time.Duration
	for _, d := range delays {
		sum += d
	}
	require.True(t, sum >= 21*time.Second && sum <= 42*time.Second, "total wait %s", sum)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 10)
	for i, line := range lines {
		require.Contains(t, line, fmt.Sprintf(" attempt=%d ", i+1))
	}
}

func TestRetryCredentialsEndpoint(t *testing.T) {
	isolate(t)
	buf := &bytes.Buffer{}
	useRequestPath(t, buf)

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

	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", srv.URL+"/creds")

	var delays []time.Duration
	_, err := ecsClient(t, srv.URL, &delays).DescribeClusters(&ecs.DescribeClustersInput{})
	require.NoError(t, err)
	require.EqualValues(t, 2, atomic.LoadInt32(&creds))
	require.Empty(t, delays)
	require.Equal(t, 1, strings.Count(buf.String(), "\n"))
	require.True(t, strings.HasPrefix(buf.String(), "ns=aws at=retry service=CredentialsEndpoint operation=GetCredentials code=SerializationError status=429 attempt=1 delay="), buf.String())
}

func TestDefaultProfileDoesNotLog(t *testing.T) {
	isolate(t)

	_, ok := sessionRetryer.(loggedRetryer)
	require.False(t, ok)

	buf := &bytes.Buffer{}
	old := logger.Output
	logger.Output = buf
	t.Cleanup(func() { logger.Output = old })

	srv, _ := throttleServer(t, 1)

	var delays []time.Duration
	_, err := ecsClient(t, srv.URL, &delays).DescribeClusters(&ecs.DescribeClustersInput{})
	require.NoError(t, err)
	require.Len(t, delays, 1)
	require.True(t, delays[0] >= 10*time.Second && delays[0] < 20*time.Second, "first wait %s", delays[0])
	require.Empty(t, buf.String())
}
