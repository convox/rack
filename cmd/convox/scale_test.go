package main

import (
	"testing"

	"github.com/convox/rack/test"
)

func TestScaleCmd(t *testing.T) {
	tests := []test.ExecRun{
		test.ExecRun{
			// Ensure we don't segfault if user is not logged in
			Command: "convox scale",
			Env:     configlessEnv,
			Exit:    0,
			Stderr:  "ERROR: no host config found, try `convox login`\n",
		},
	}
	for _, myTest := range tests {
		test.Runs(t, myTest)
	}
}
