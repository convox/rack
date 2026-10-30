package aws

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/convox/rack/pkg/crypt"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

type sharedStub struct {
	*httptest.Server
	gets  int32
	short atomic.Bool

	mu   sync.Mutex
	auth []string
}

func newSharedStub(t *testing.T) *sharedStub {
	st := &sharedStub{}

	st.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/creds" {
			atomic.AddInt32(&st.gets, 1)
			valid := 6 * time.Hour
			if st.short.CompareAndSwap(true, false) {
				valid = 8 * time.Minute
			}
			_, _ = fmt.Fprintf(w, `{"AccessKeyId":"shared","SecretAccessKey":"b","Token":"c","Expiration":%q}`, time.Now().Add(valid).UTC().Format(time.RFC3339))
			return
		}

		st.mu.Lock()
		st.auth = append(st.auth, r.Header.Get("Authorization"))
		st.mu.Unlock()

		body, _ := io.ReadAll(r.Body)
		target := r.Header.Get("X-Amz-Target")

		switch {
		case strings.HasSuffix(target, "GetAuthorizationToken"):
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = fmt.Fprintf(w, `{"authorizationData":[{"authorizationToken":%q}]}`, base64.StdEncoding.EncodeToString([]byte("AWS:token")))
		case strings.HasSuffix(target, "DescribeLogGroups"):
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = fmt.Fprint(w, `{"logGroups":[]}`)
		case strings.HasSuffix(target, "GenerateRandom"):
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = fmt.Fprintf(w, `{"Plaintext":%q}`, base64.StdEncoding.EncodeToString(make([]byte, 24)))
		case strings.Contains(string(body), "Action=ListStackResources"):
			_, _ = fmt.Fprint(w, `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/"><ListStackResourcesResult><StackResourceSummaries><member><LogicalResourceId>Settings</LogicalResourceId><PhysicalResourceId>settings</PhysicalResourceId><ResourceType>AWS::S3::Bucket</ResourceType><ResourceStatus>CREATE_COMPLETE</ResourceStatus></member></StackResourceSummaries></ListStackResourcesResult></ListStackResourcesResponse>`)
		case r.Method == "PUT":
			w.Header().Set("ETag", `"x"`)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(st.Close)

	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", st.URL+"/creds")

	helpers.UseSharedCredentials()

	t.Cleanup(func() {
		require.NoError(t, os.Setenv("AWS_ACCESS_KEY_ID", "test"))
		helpers.UseSharedCredentials()
	})

	return st
}

func (st *sharedStub) credentialGets() int32 {
	return atomic.LoadInt32(&st.gets)
}

func TestSharedCredentialsBypasses(t *testing.T) {
	st := newSharedStub(t)
	p := &Provider{Region: "us-east-1", Endpoint: st.URL, Rack: "convox", SkipCache: true}
	host := "123456789012.dkr.ecr.us-east-1.amazonaws.com"

	for i := 0; i < 3; i++ {
		_, _, err := p.authECR(host, "", "")
		require.NoError(t, err)
	}

	_, err := p.cwlogs().DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{})
	require.NoError(t, err)

	c := crypt.New()
	c.Config = helpers.SessionConfig().WithEndpoint(st.URL)

	_, err = crypt.KMS(c).GenerateRandom(&kms.GenerateRandomInput{NumberOfBytes: aws.Int64(24)})
	require.NoError(t, err)

	require.EqualValues(t, 1, st.credentialGets())

	_, _, err = p.authECR(host, "AKIDSTATIC", "secret")
	require.NoError(t, err)
	require.EqualValues(t, 1, st.credentialGets())

	st.mu.Lock()
	defer st.mu.Unlock()
	require.Contains(t, st.auth[len(st.auth)-1], "Credential=AKIDSTATIC/")
	require.Contains(t, st.auth[0], "Credential=shared/")
}

func TestObjectStorePresignRefetch(t *testing.T) {
	st := newSharedStub(t)
	p := &Provider{Region: "us-east-1", Endpoint: st.URL, Rack: "convox", SkipCache: true}

	store := func() {
		_, err := p.ObjectStore("", "k", bytes.NewReader([]byte("x")), structs.ObjectStoreOptions{Presign: options.Bool(true)})
		require.NoError(t, err)
	}

	for i := 0; i < 20; i++ {
		store()
	}
	require.EqualValues(t, 1, st.credentialGets())

	st.short.Store(true)
	helpers.UseSharedCredentials()

	store()
	require.EqualValues(t, 3, st.credentialGets())

	store()
	require.EqualValues(t, 3, st.credentialGets())
}

func TestS3StreamingClient(t *testing.T) {
	newSharedStub(t)
	p := &Provider{Region: "us-east-1"}

	require.NotNil(t, helpers.StreamingHTTPClient())
	require.Same(t, helpers.StreamingHTTPClient(), p.s3Streaming().Config.HTTPClient)
	require.NotSame(t, helpers.StreamingHTTPClient(), p.s3().Config.HTTPClient)
}
