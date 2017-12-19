package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

// TestServices verifies that resources can still be listed via the 'convox services' command (for backwards compatibility).
func TestServices(t *testing.T) {
	ts := testServer(t,
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

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox services",
			Exit:    0,
			Stdout:  "NAME         TYPE    STATUS\nsyslog-1234  syslog  running\n",
		},
	)
}

// TestServicesGet verifies that resources can still be retrieved via the 'convox services info' command (for backwards compatibility).
func TestServicesGet(t *testing.T) {
	ts := testServer(t,
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

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox services info syslog-1234",
			Exit:    0,
			Stdout:  "Name    syslog-1234\nStatus  running\n",
		},
	)
}

// TestServicesCreate verifies that resources can still be created via the 'convox services create' command (for backwards compatibility).
func TestServicesCreate(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "POST",
			Path:     "/resources",
			Body:     "name=syslog-1234&type=syslog&url=tcp%2Btls%3A%2F%2Flogs1.example.com%3A12345",
			Code:     200,
			Response: client.Resource{},
		},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox services create syslog --name=syslog-1234 --url=tcp+tls://logs1.example.com:12345",
			Exit:    0,
			Stdout:  "Creating syslog-1234 (syslog: name=\"syslog-1234\" url=\"tcp+tls://logs1.example.com:12345\")... CREATING\n",
		},
	)
}

// TestServicesUpdate verifies that a resource can be still updated via the 'convox services update' command (for backwards compatibility).
func TestServicesUpdate(t *testing.T) {
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

// TestServicesDelete verifies that resources can still be deleted via the 'convox services delete' command (for backwards compatibility).
func TestServicesDelete(t *testing.T) {
	tsd := testServer(t,
		test.Http{
			Method:   "DELETE",
			Path:     "/resources/syslog-1234",
			Code:     200,
			Response: client.Resource{},
		},
	)

	defer tsd.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox services delete syslog-1234",
			Exit:    0,
			Stdout:  "Deleting syslog-1234... DELETING\n",
		},
	)
}

// func TestWaitForResource(t *testing.T) {
//   err := testWaitForResource(t, "running", false)
//   require.NoError(t, err)

//   err = testWaitForResource(t, "running", true)
//   require.NoError(t, err)

//   err = testWaitForResource(t, "updating", true)
//   assert.EqualError(t, err, "timeout")

//   err = testWaitForResource(t, "rollback", true)
//   assert.EqualError(t, err, "timeout")
// }

func testWaitForResource(t *testing.T, s string, w bool) error {
	ts := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/resources/mysql-db",
			Code:   200,
			Response: client.Resource{
				Name:   "mysql-db",
				Status: s,
				Type:   "mysql",
			},
		},
	)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)

	c := client.New(u.Host, "test", "test")

	waitSecond = time.Millisecond
	return waitForResource(c, "mysql-db", "CREATING", w)
}
