package main

import (
	"testing"

	"github.com/convox/rack/test"
)

var scaleUsage = `convox scale: scale an app's processes`

func TestScaleCmd(t *testing.T) {

	ts := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/apps/convox/processes",
			Code:   200,
		},
		test.Http{
			Method: "GET",
			Path:   "/apps/convox/formation",
			Code:   200,
		},
		test.Http{
			Method: "POST",
			Path:   "/apps/convox/formation/myprocesses",
			Body:   "cpu=1",
			Code:   200,
		},
	)
	defer ts.Close()

	tests := []test.ExecRun{
		test.ExecRun{
			Command:  "convox scale foo bar",
			OutMatch: scaleUsage,
			Exit:     129,
		},
		test.ExecRun{
			Command:  "convox scale --foo",
			OutMatch: "Incorrect Usage: flag provided but not defined: -foo\n\n" + scaleUsage,
			Stderr:   "ERROR: flag provided but not defined: -foo\n",
			Exit:     1,
		},
		test.ExecRun{
			Command:  "convox scale --cpu",
			OutMatch: "Incorrect Usage: flag needs an argument: -cpu\n\n" + scaleUsage,
			Stderr:   "ERROR: flag needs an argument: -cpu\n",
			Exit:     1,
		},
		test.ExecRun{
			Command: "convox scale --cpu=1",
			Stderr:  "ERROR: missing process name\n",
			Exit:    1,
		},
		test.ExecRun{
			Command:  "convox scale --cpu=1 myprocesses",
			OutMatch: "NAME  DESIRED  RUNNING  CPU  MEMORY\n",
		},
	}

	for _, myTest := range tests {
		test.Runs(t, myTest)
	}
}
