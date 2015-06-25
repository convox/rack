package main

import (
	"bytes"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func appRun(args []string) (string, string) {
	app := stdcli.New()
	stdcli.Exiter = func(code int) {}

	// Capture stdout and stderr to strings via Pipes
	oldErr := os.Stderr
	oldOut := os.Stdout

	er, ew, _ := os.Pipe()
	or, ow, _ := os.Pipe()

	os.Stderr = ew
	os.Stdout = ow

	errC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, er)
		errC <- buf.String()
	}()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, or)
		outC <- buf.String()
	}()

	_ = app.Run(args)

	// restore stderr, stdout
	ew.Close()
	os.Stderr = oldErr
	err := <-errC

	ow.Close()
	os.Stdout = oldOut
	out := <-outC

	return out, err
}

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

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
