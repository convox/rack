package main

import (
	"testing"

	"github.com/convox/rack/test"
)

func TestUninstall(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/foo",
			Code:     200,
			Response: "bar",
			Headers:  map[string]string{"Rack": "myorg/staging"},
		},
	)
	defer ts.Close()

	tests := []test.ExecRun{
		// help flags
		test.ExecRun{
			Command:  "convox uninstall -h",
			OutMatch: "convox uninstall: uninstall a convox rack",
		},

		test.ExecRun{
			Command:  "convox uninstall",
			OutMatch: "convox uninstall: uninstall a convox rack",
			Exit:     129,
		},
		test.ExecRun{
			Command:  "convox uninstall onlyOneArgument",
			OutMatch: "convox uninstall: uninstall a convox rack",
			Exit:     129,
		},
		test.ExecRun{
			Command:  "convox uninstall more than three arguments",
			OutMatch: "convox uninstall: uninstall a convox rack",
			Exit:     129,
		},
	}

	for _, myTest := range tests {
		test.Runs(t, myTest)
	}
}
