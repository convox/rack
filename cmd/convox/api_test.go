package main

import (
	"testing"

	"github.com/convox/rack/test"
)

func TestApiGet(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/foo", Code: 200, Response: "bar"},
	)
	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox api get /foo",
			Exit:    0,
			Stdout:  "\"bar\"\n",
		},
	)
}

func TestRackFlag(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/foo", Code: 200, Response: "bar", Headers: map[string]string{"Rack": "myorg/staging"}},
	)
	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command:  "convox --rack myorg/staging api get /foo",
			Exit:     0,
			OutMatch: "bar",
		},
	)

	test.Runs(t,
		test.ExecRun{
			Command:  "convox api --rack myorg/staging get /foo",
			Exit:     0,
			OutMatch: "bar",
		},
	)

	test.Runs(t,
		test.ExecRun{
			Command:  "convox api get --rack myorg/staging /foo",
			Exit:     0,
			OutMatch: "bar",
		},
	)

	test.Runs(t,
		test.ExecRun{
			Command:  "convox api get /foo --rack myorg/staging",
			Exit:     0,
			OutMatch: "bar",
		},
	)
}
