package main

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
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

	expect(t, stdout, "Login Succeeded\n")
	expect(t, stderr, "")
}

func TestLoginHost(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ok", 200)
	}))
	defer ts.Close()

	stdout, stderr := appRun([]string{"convox", "login", "--password", "foobar", ts.URL})

	expect(t, stdout, "Login Succeeded\n")
	expect(t, stderr, "")

	u, _ := url.Parse(ts.URL)
	stdout, stderr = appRun([]string{"convox", "login", "--password", "foobar", u.Host})

	expect(t, stdout, "Login Succeeded\n")
	expect(t, stderr, "")

	stdout, stderr = appRun([]string{"convox", "login", "--password", "foobar", "BAD"})

	expect(t, stdout, "")
	expect(t, stderr, "ERROR: Get https://BAD/apps: dial tcp: lookup BAD: no such host\n")
}

func TestCommandDoNotIgnoreFlags(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	test := []string{"blah", "blah", "-break"}
	set.Parse(test)

	c := cli.NewContext(app, set, nil)

	command := cli.Command{
		Name:        "test-cmd",
		Aliases:     []string{"tc"},
		Usage:       "this is for testing",
		Description: "testing",
		Action:      func(_ *cli.Context) {},
	}
	err := command.Run(c)

	expect(t, err.Error(), "flag provided but not defined: -break")
}
