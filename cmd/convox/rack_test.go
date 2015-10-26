package main

import (
	"fmt"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/convox/release/version"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/require"
	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestRackUpdateStable(t *testing.T) {
	versions, err := version.All()
	require.Nil(t, err)

	stable, err := versions.Resolve("stable")
	require.Nil(t, err)

	ts := testServer(t,
		test.Http{Method: "PUT", Body: fmt.Sprintf("version=%s", stable.Version), Path: "/system", Code: 200, Response: client.System{
			Name:    "mysystem",
			Version: "ver",
			Count:   1,
			Type:    "type",
		}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox rack update",
			Exit:    0,
			Stdout:  "Name     mysystem\nStatus   \nVersion  ver\nCount    1\nType     type\n",
		},
	)
}

func TestRackUpdateSpecified(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "PUT", Body: "version=20150909014908", Path: "/system", Code: 200, Response: client.System{
			Name:    "mysystem",
			Version: "ver",
			Count:   1,
			Type:    "type",
		}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox rack update 20150909014908",
			Exit:    0,
			Stdout:  "Name     mysystem\nStatus   \nVersion  ver\nCount    1\nType     type\n",
		},
	)
}
