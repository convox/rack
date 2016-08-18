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

// new http tester

type HandlerFuncTest struct {
	Handler http.HandlerFunc

	body []byte
	code int
}

func NewHandlerFunc(handler http.HandlerFunc) HandlerFuncTest {
	return HandlerFuncTest{
		Handler: handler,
	}
}

func (f *HandlerFuncTest) Request(method, url string, values url.Values) error {
	w := httptest.NewRecorder()

	req, err := buildRequest(method, url, values)
	if err != nil {
		return err
	}

	f.Handler(w, req)

	f.body = w.Body.Bytes()
	f.code = w.Code

	return nil
}

func (f *HandlerFuncTest) AssertCode(t *testing.T, code int) {
	assert.Equal(t, code, f.code)
}

func (f *HandlerFuncTest) AssertError(t *testing.T, message string) {
	var err struct {
		Error string `json:"error"`
	}

	if assert.Nil(t, json.Unmarshal(f.Body(), &err)) {
		assert.Equal(t, message, err.Error)
	}
}

func (f *HandlerFuncTest) AssertJSON(t *testing.T, body string) {
	b1, err1 := stripJson([]byte(body))
	b2, err2 := stripJson(f.Body())

	if assert.Nil(t, err1) && assert.Nil(t, err2) {
		assert.Equal(t, b1, b2)
	}
}

func (f *HandlerFuncTest) Body() []byte {
	return f.body
}

func (f *HandlerFuncTest) Code() int {
	return f.code
}

func stripJson(data []byte) ([]byte, error) {
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
