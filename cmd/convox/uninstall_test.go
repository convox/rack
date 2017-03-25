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
			Command: "convox uninstall foo us-east-1",
			Env: map[string]string{
				"AWS_ACCESS_KEY_ID":     "",
				"AWS_SECRET_ACCESS_KEY": "",
			},
			Exit:     0,
			OutMatch: "This installer needs AWS credentials to install/uninstall the Convox platform",
		},

		// FIXME: this command actually exits with a 'no such file or directory' error
		test.ExecRun{
			Command:  "convox uninstall rackArg regionArg credentialsArg",
			OutMatch: Banner + "\nReading credentials from file credentialsArg\n",
			Exit:     0,
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
