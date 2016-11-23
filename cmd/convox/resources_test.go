package main

import (
	"testing"

    "github.com/convox/rack/client"
    "github.com/convox/rack/test"
)

func TestResources(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/services", Code: 200, Response: client.Services{
			client.Service{Name: "syslog-1234", Type: "syslog", Status: "running"},
		}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox resources",
			Exit:	 0,
			Stdout:	 "NAME         TYPE    STATUS\nsyslog-1234  syslog  running\n",
		},
	)
}

func TestResourcesCreate(t *testing.T) {
    ts := testServer(t,
        test.Http{Method: "POST", Path: "/services", Body: "name=syslog-1234&type=syslog&url=tcp%2Btls%3A%2F%2Flogs1.example.com%3A12345", Code: 200, Response: client.Service{}},
    )

    defer ts.Close()

    test.Runs(t,
        test.ExecRun{
            Command: "convox resources create syslog --name=syslog-1234 --url=tcp+tls://logs1.example.com:12345",
            Exit:    0,
            Stdout:  "Creating syslog-1234 (syslog: name=\"syslog-1234\" url=\"tcp+tls://logs1.example.com:12345\")... CREATING\n",
        },
    )
}

func TestResourcesDelete(t *testing.T) {
    tsd := testServer(t,
        test.Http{Method: "DELETE", Path: "/services/syslog-1234", Code: 200, Response: client.Service{}},
    )

    defer tsd.Close()

    test.Runs(t,
        test.ExecRun{
            Command: "convox resources delete syslog-1234",
            Exit:    0,
            Stdout:  "Deleting syslog-1234... DELETING\n",
        },
    )
}

