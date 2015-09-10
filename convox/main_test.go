package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/convox/cli/client"
	"github.com/convox/cli/stdcli"
	"github.com/stretchr/testify/assert"
)

type Run struct {
	Command []string
	Stdout  string
	Stderr  string
}

type Stub struct {
	Method   string
	Path     string
	Code     int
	Response interface{}
}

func init() {
	dir, _ := ioutil.TempDir("", "convox-test")
	os.Setenv("CONVOX_CONFIG", dir)
}

func testRuns(t *testing.T, ts *httptest.Server, runs ...Run) {
	u, _ := url.Parse(ts.URL)

	os.Setenv("CONVOX_HOST", u.Host)

	for _, run := range runs {
		stdout, stderr := appRun(run.Command)

		assert.Equal(t, run.Stdout, stdout, "stdout should be equal")
		assert.Equal(t, run.Stderr, stderr, "stderr should be equal")
	}
}

func httpStub(stubs ...Stub) *httptest.Server {
	stubs = append(stubs, Stub{Method: "GET", Path: "/system", Code: 200, Response: client.System{
		Version: "latest",
	}})

	found := false

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, stub := range stubs {
			if stub.Method == r.Method && stub.Path == r.URL.Path {
				data, err := json.Marshal(stub.Response)

				if err != nil {
					http.Error(w, err.Error(), 503)
				}

				w.WriteHeader(stub.Code)
				w.Write(data)

				found = true
				break
			}
		}

		if !found {
			fmt.Printf("unknown request: %+v\n", r)
			http.Error(w, "not found", 404)
		}
	}))

	u, _ := url.Parse(ts.URL)

	dir, _ := ioutil.TempDir("", "convox-test")

	ConfigRoot, _ = ioutil.TempDir("", "convox-test")

	os.Setenv("CONVOX_CONFIG", dir)
	os.Setenv("CONVOX_HOST", u.Host)
	os.Setenv("CONVOX_PASSWORD", "foo")

	return ts
}

func appRun(args []string) (string, string) {
	app := stdcli.New()
	stdcli.Exiter = func(code int) {}
	stdcli.Runner = func(bin string, args ...string) error { return nil }
	stdcli.Querier = func(bin string, args ...string) ([]byte, error) { return []byte{}, nil }
	stdcli.Tagger = func() string { return "1435444444" }
	stdcli.Writer = func(filename string, data []byte, perm os.FileMode) error { return nil }

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
