package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

// TestServices verifies that resources can be listed via the 'convox resources' command.
func TestResources(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/resources",
			Code:   200,
			Response: client.Resources{
				client.Resource{
					Name:   "syslog-1234",
					Type:   "syslog",
					Status: "running",
				},
			},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox resources",
			Exit:    0,
			Stdout:  "NAME         TYPE    STATUS\nsyslog-1234  syslog  running\n",
		},
	)
}

// TestResourcesGet verifies that resources can be retrieved via the 'convox resources info' command.
func TestResourcesGet(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/resources/syslog-1234",
			Code:   200,
			Response: client.Resource{
				Name:   "syslog-1234",
				Type:   "syslog",
				Status: "running",
			},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox resources info syslog-1234",
			Exit:    0,
			Stdout:  "Name    syslog-1234\nStatus  running\n",
		},
	)
}

// TestResourcesCreate verifies that resources can be created via the 'convox resources create' command.
func TestResourcesCreate(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method:   "POST",
			Path:     "/resources",
			Body:     "name=syslog-1234&type=syslog&url=tcp%2Btls%3A%2F%2Flogs1.example.com%3A12345",
			Code:     200,
			Response: client.Resource{},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox resources create syslog --name=syslog-1234 --url=tcp+tls://logs1.example.com:12345",
			Exit:    0,
			Stdout:  "Creating syslog-1234 (syslog: name=\"syslog-1234\" url=\"tcp+tls://logs1.example.com:12345\")... CREATING\n",
		},
	)
}

// TestResourcesUpdate verifies that a resource can be updated via the 'convox resources update' command.
func TestResourcesUpdate(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "PUT",
			Path:   "/resources/syslog-1234",
			Body:   "url=tcp%2Btls%3A%2F%2Flogs1.example.net%3A12345",
			Code:   200,
			Response: client.Resource{
				Name:   "syslog-1234",
				Status: "updating",
			},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox resources update syslog-1234 --url=tcp+tls://logs1.example.net:12345",
			Exit:    0,
			Stdout:  "Updating syslog-1234 (url=\"tcp+tls://logs1.example.net:12345\")...UPDATING\n",
		},
	)
}

// TestResourcesDelete verifies that resources can be deleted via the 'convox resources delete' command.
func TestResourcesDelete(t *testing.T) {
	trd := testServer(t,
		test.Http{
			Method:   "DELETE",
			Path:     "/resources/syslog-1234",
			Code:     200,
			Response: client.Resource{},
		},
	)

	defer trd.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox resources delete syslog-1234",
			Exit:    0,
			Stdout:  "Deleting syslog-1234... DELETING\n",
		},
	)
}
