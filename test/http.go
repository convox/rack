package test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Http struct {
	Method   string
	Path     string
	Code     int
	Body     string
	Response interface{}
}

var HandlerFunc http.HandlerFunc

func AssertStatus(t *testing.T, status int, method, url string, values url.Values) string {
	w := httptest.NewRecorder()
	req, err := buildRequest(method, url, values)

	if err != nil {
		t.Error(err)
		return ""
	}

	HandlerFunc(w, req)
	assert.Equal(t, status, w.Code)
	return w.Body.String()
}

func HTTPBody(method, url string, values url.Values) string {
	w := httptest.NewRecorder()
	req, err := buildRequest(method, url, values)

	if err != nil {
		return ""
	}

	HandlerFunc(w, req)
	return w.Body.String()
}

func buildRequest(method, url string, values url.Values) (req *http.Request, err error) {

	if method == "POST" {
		postBody := strings.NewReader(values.Encode())
		req, err = http.NewRequest("POST", url, postBody)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest(method, url+"?"+values.Encode(), nil)
	}
	req.Header.Set("Version", "dev")

	return
}
