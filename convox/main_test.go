package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/convox/cli/stdcli"
)

func appRun(args []string) (string, string) {
	app := stdcli.New()
	stdcli.Exiter = func(code int) {}
	stdcli.Runner = func(bin string, args ...string) error { return nil }
	stdcli.Querier = func(bin string, args ...string) ([]byte, error) { return []byte{}, nil }

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

func setLoginEnv(ts *httptest.Server) {
	u, _ := url.Parse(ts.URL)
	os.Setenv("CONSOLE_HOST", u.Host)
	os.Setenv("REGISTRY_PASSWORD", "foo")
}

func expect(t *testing.T, a interface{}, b interface{}) {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)

	if !bytes.Equal(aj, bj) {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
