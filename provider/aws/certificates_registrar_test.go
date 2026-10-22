package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	certbotAuthorization = regexp.MustCompile(`^AWS4-HMAC-SHA256 Credential=[A-Za-z0-9/_-]+, SignedHeaders=([a-z0-9;-]+), Signature=[0-9a-f]{64}$`)
	certbotSTS           = regexp.MustCompile(`^https://sts(\.[a-z]{2}(-gov)?-[a-z]+-[0-9])?\.amazonaws\.com/$`)
	certbotSigned        = map[string]bool{"content-length": true, "content-type": true, "host": true, "x-amz-date": true, "x-amz-security-token": true, "x-convox-registrar": true}
)

// TestRegistrarIdentity applies the checks certbot's identify makes before it calls STS.
func TestRegistrarIdentity(t *testing.T) {
	for _, region := range []string{"us-east-1", "ap-east-1", "us-gov-west-1"} {
		t.Run(region, func(t *testing.T) {
			t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
			t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
			t.Setenv("AWS_SESSION_TOKEN", "token")
			t.Setenv("AWS_REGION", region)

			p := &Provider{Region: region}

			id, err := p.registrarIdentity()
			require.NoError(t, err)

			data, err := json.Marshal(registrarRequest{Identity: *id})
			require.NoError(t, err)

			var in struct {
				Identity struct {
					URL     string              `json:"url"`
					Headers map[string][]string `json:"headers"`
					Body    string              `json:"body"`
				} `json:"identity"`
			}
			require.NoError(t, json.Unmarshal(data, &in))

			assert.Regexp(t, certbotSTS, in.Identity.URL)
			assert.Equal(t, "Action=GetCallerIdentity&Version=2011-06-15", in.Identity.Body)

			h := http.Header{}
			for k, v := range in.Identity.Headers {
				ck := http.CanonicalHeaderKey(k)
				_, dup := h[ck]
				require.False(t, dup, "header %s under two spellings", k)
				h[ck] = v
			}

			assert.Equal(t, []string{"convox.site"}, h["X-Convox-Registrar"])
			require.Len(t, h["Authorization"], 1)

			m := certbotAuthorization.FindStringSubmatch(h.Get("Authorization"))
			require.NotNil(t, m, h.Get("Authorization"))

			signed := strings.Split(m[1], ";")
			for _, s := range signed {
				assert.True(t, certbotSigned[s], "signed header %s", s)
			}
			assert.Subset(t, signed, []string{"host", "x-amz-date", "x-convox-registrar", "x-amz-security-token"})
		})
	}
}

func TestRegistrarPublish(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_REGION", "us-east-1")

	var got map[string]interface{}
	var contentType string

	status, reply := 200, `{"name":"n","value":"v"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)

	saved := certificateRegistrarURL
	certificateRegistrarURL = srv.URL
	t.Cleanup(func() { certificateRegistrarURL = saved })

	p := &Provider{Region: "us-east-1"}

	require.NoError(t, p.registrarPublish(p.Context(), "*.lb.us-east-1.convox.site", "_n.lb.us-east-1.convox.site.", "_v.x.acm-validations.aws."))
	assert.Equal(t, "application/json", contentType)
	assert.Equal(t, "*.lb.us-east-1.convox.site", got["domain"])
	assert.Equal(t, "_n.lb.us-east-1.convox.site.", got["name"])
	assert.Equal(t, "_v.x.acm-validations.aws.", got["value"])
	assert.IsType(t, "", got["identity"].(map[string]interface{})["url"])

	status, reply = 403, `{"error":"load balancer belongs to another account"}`
	err := p.registrarPublish(p.Context(), "*.lb.us-east-1.convox.site", "_n.lb.us-east-1.convox.site.", "_v.x.acm-validations.aws.")
	assert.EqualError(t, err, "registrar 403: load balancer belongs to another account")

	status, reply = 502, `bad gateway`
	err = p.registrarPublish(p.Context(), "*.lb.us-east-1.convox.site", "_n.lb.us-east-1.convox.site.", "_v.x.acm-validations.aws.")
	assert.EqualError(t, err, "registrar 502: bad gateway")
	assert.True(t, err.(registrarError).retry())
}
