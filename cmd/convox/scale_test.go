package main

import (
	"testing"

	"github.com/convox/rack/test"
)

var scaleUsage = `convox scale: scale an app's processes

Usage:
  convox scale <process> [--count=2] [--memory=256] [--cpu=256]

Options:
   --app value, -a value  app name inferred from current directory if not specified
   --rack value           rack name
   --count value          Number of processes to keep running for specified process type. (default: 0)
   --memory value         Amount of memory, in MB, available to specified process type. (default: 0)
   --cpu value            CPU units available to specified process type. (default: 0)
   --wait                 wait for app to finish scaling before returning [$CONVOX_WAIT]
   
`

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
			Command: "convox scale",
			Stdout:  "NAME  DESIRED  RUNNING  CPU  MEMORY\n",
			Exit:    0,
		},
		test.ExecRun{
			Command: "convox scale foo bar",
			Stdout:  scaleUsage,
			Exit:    129,
		},
		test.ExecRun{
			Command: "convox scale --foo",
			Stdout:  "Incorrect Usage: flag provided but not defined: -foo\n\n" + scaleUsage,
			Stderr:  "ERROR: flag provided but not defined: -foo\n",
			Exit:    1,
		},
		test.ExecRun{
			Command: "convox scale --cpu",
			Stdout:  "Incorrect Usage: flag needs an argument: -cpu\n\n" + scaleUsage,
			Stderr:  "ERROR: flag needs an argument: -cpu\n",
			Exit:    1,
		},
		test.ExecRun{
			Command: "convox scale --cpu=1",
			Stderr:  "ERROR: missing process name\n",
			Exit:    1,
		},
		test.ExecRun{
			Command: "convox scale --cpu=1 myprocesses",
			Stdout:  "NAME  DESIRED  RUNNING  CPU  MEMORY\n",
		},
	}

	for _, myTest := range tests {
		test.Runs(t, myTest)
	}
}
