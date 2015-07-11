package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestInvalidLogin(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", 401)
	}))
	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "")
	expect(t, stderr, "ERROR: invalid login\n")
}

func TestLogin(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", 200)
	}))
	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "Logged in successfully.\n")
	expect(t, stderr, "")
}

func TestLoginHost(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", 200)
	}))
	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "Logged in successfully.\n")
	expect(t, stderr, "")

	u, _ := url.Parse(ts.URL)
	stdout, stderr = appRun([]string{"convox", "login", "--password", "foobar", u.Host})

	expect(t, stdout, "Logged in successfully.\n")
	expect(t, stderr, "")

	stdout, stderr = appRun([]string{"convox", "login", "--password", "foobar", "BAD"})

	expect(t, stdout, "")
	expect(t, stderr, "ERROR: Get https://BAD/apps: dial tcp: lookup BAD: no such host\n")
}
