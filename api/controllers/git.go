package controllers

import (
	"bytes"
	hm "crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/gorilla/mux"
)

var regexpCC = regexp.MustCompile(`git-codecommit\.([^.]+)\.amazonaws\.com.*`)

// Proxy to a HTTP git service. Tested with CodeCommit and GitHub
// Set git debugging variables to inspect push/pull traffic:
// export GIT_TRACE_PACKET=1
// export GIT_TRACE=1
// export GIT_CURL_VERBOSE=1
func GitProxy(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse("https://git-codecommit.us-east-1.amazonaws.com/v1/repos/httpd")

	// rewrite request Host and Auth headers
	r.Host = u.Host
	r.SetBasicAuth(os.Getenv("AWS_ACCESS"), credentialHelper(u))

	// rewrite request Path to remove /apps/{app}/repo
	r.URL.Path = mux.Vars(r)["rest"]

	// reverse proxy to HTTP git service
	// Debug by setting a Transport with request/response logging like https://github.com/motemen/go-loghttp
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(w, r)
}

// Implements codecommit credential-helper:
//   https://github.com/aws/aws-cli/blob/develop/awscli/customizations/codecommit.py#L145
//   $ echo -e "protocol=https\npath=v1/repos/httpd\nhost=git-codecommit.us-east-1.amazonaws.com" | AWS_ACCESS_KEY_ID=AKIAIQF3QUHATF2AY3FQ AWS_SECRET_ACCESS_KEY=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 aws codecommit credential-helper get
//   username=AKIAIQF3QUHATF2AY3FQ
//   password=20160227T172057Z56238a5ac75c8bd36bba91737377aca46c867f584a6695f0486f3e3bba9b4ed5
// References:
//   https://github.com/crowdmob/goamz/blob/master/aws/sign.go
func credentialHelper(u *url.URL) string {
	t := time.Now()
	// t, _ := time.Parse("20060102T150405", "20160227T172057") // reference time the comments were generated with

	region := os.Getenv("AWS_REGION")

	if match := regexpCC.FindStringSubmatch(u.Host); len(match) > 1 {
		region = match[1]
	}

	// Build canonical request
	// 'GIT\n/v1/repos/httpd\n\nhost:git-codecommit.us-east-1.amazonaws.com\n\nhost\n'
	cr := new(bytes.Buffer)
	fmt.Fprintf(cr, "%s\n", "GIT")         // HTTPRequestMethod
	fmt.Fprintf(cr, "%s\n", u.Path)        // CanonicalURI
	fmt.Fprintf(cr, "%s\n", "")            // CanonicalQueryString
	fmt.Fprintf(cr, "host:%s\n\n", u.Host) // CanonicalHeaders
	fmt.Fprintf(cr, "%s\n", "host")        // SignedHeaders
	fmt.Fprintf(cr, "%s", "")              // HexEncode(Hash(Payload))

	// Build string to sign
	// 'AWS4-HMAC-SHA256\n20160227T172057\n20160227/us-east-1/codecommit/aws4_request\n650b9e2de2abce7c30f6ad51c4a84b361e1f8aaaa3152e93d35509450db2d869'
	sts := new(bytes.Buffer)
	fmt.Fprint(sts, "AWS4-HMAC-SHA256\n")                                                   // Algorithm
	fmt.Fprintf(sts, "%s\n", t.Format("20060102T150405"))                                   // RequestDate
	fmt.Fprintf(sts, "%s/%s/%s/aws4_request\n", t.Format("20060102"), region, "codecommit") // CredentialScope
	fmt.Fprintf(sts, "%s", hash(cr.String()))                                               // HexEncode(Hash(CanonicalRequest))

	// Calculate the AWS Signature Version 4
	// '56238a5ac75c8bd36bba91737377aca46c867f584a6695f0486f3e3bba9b4ed5'
	dsk := hmac([]byte("AWS4"+os.Getenv("AWS_SECRET")), []byte(t.Format("20060102")))
	dsk = hmac(dsk, []byte(region))
	dsk = hmac(dsk, []byte("codecommit"))
	dsk = hmac(dsk, []byte("aws4_request"))
	h := hmac(dsk, []byte(sts.String()))
	sig := fmt.Sprintf("%x", h) // HexEncode(HMAC(derived-signing-key, string-to-sign))

	// codecommmit smart http password to use with AWS_ACCESS
	return fmt.Sprintf("%sZ%s", t.Format("20060102T150405"), sig)
}

// hash method calculates the sha256 hash for a given string
func hash(in string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s", in)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// hmac method calculates the sha256 hmac for a given slice of bytes
func hmac(key, data []byte) []byte {
	h := hm.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
