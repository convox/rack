package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// HandlerFuncTest is a helper for running tests on http.HandlerFunc
type HandlerFuncTest struct {
	Handler http.HandlerFunc

	body    []byte
	code    int
	version string
}

// NewHandlerFunc returns a new HandlerFuncTest
func NewHandlerFunc(handler http.HandlerFunc) HandlerFuncTest {
	return HandlerFuncTest{
		Handler: handler,
		version: "dev",
	}
}

// Request executes an HTTP request against the tester
func (f *HandlerFuncTest) Request(method, url string, values url.Values) error {
	w := httptest.NewRecorder()

	req, err := f.request(method, url, values)
	if err != nil {
		return err
	}

	f.Handler(w, req)

	f.body = w.Body.Bytes()
	f.code = w.Code

	return nil
}

// AssertCode asserts the response code
func (f *HandlerFuncTest) AssertCode(t *testing.T, code int) {
	assert.Equal(t, code, f.code)
}

// AssertError asserts a response error
func (f *HandlerFuncTest) AssertError(t *testing.T, message string) {
	var err struct {
		Error string `json:"error"`
	}

	if assert.Nil(t, json.Unmarshal(f.Body(), &err)) {
		assert.Equal(t, message, err.Error)
	}
}

// AssertJSON assets a JSON response (ignoring whitespace differences)
func (f *HandlerFuncTest) AssertJSON(t *testing.T, body string) {
	b1, err1 := stripJSON([]byte(body))
	b2, err2 := stripJSON(f.Body())

	if assert.NoError(t, err1) && assert.NoError(t, err2) {
		assert.Equal(t, string(b1), string(b2))
	}
}

// AssertSuccess asserts a successful response
func (f *HandlerFuncTest) AssertSuccess(t *testing.T) {
	f.AssertJSON(t, `{"success":true}`)
}

// Body returns the response body
func (f *HandlerFuncTest) Body() []byte {
	return f.body
}

// Code returns the response code
func (f *HandlerFuncTest) Code() int {
	return f.code
}

// SetVersion sets the Version: HTTP header
func (f *HandlerFuncTest) SetVersion(version string) {
	f.version = version
}

func (f *HandlerFuncTest) request(method, url string, values url.Values) (req *http.Request, err error) {
	switch method {
	case "POST", "PUT":
		req, err = http.NewRequest(method, url, strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	default:
		req, err = http.NewRequest(method, url+"?"+values.Encode(), nil)
	}

	req.Header.Set("Version", f.version)

	return
}

func stripJSON(data []byte) ([]byte, error) {
	var obj interface{}

	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	strip, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return strip, nil
}
