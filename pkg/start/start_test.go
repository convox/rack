package start_test

import (
	"net/http"
	"net/http/httptest"
)

type MockHealthCheck struct {
	*httptest.Server
	count   int
	handler func(int) int
}

func (mhc *MockHealthCheck) Count() int {
	return mhc.count
}

func (mhc *MockHealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(mhc.handler(mhc.count))
	mhc.count++
}

func mockHealthCheck(fn func(n int) int) *MockHealthCheck {
	m := &MockHealthCheck{handler: fn}
	m.Server = httptest.NewTLSServer(m)
	return m
}
