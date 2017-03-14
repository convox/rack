package main

import (
	"testing"

	"github.com/convox/rack/test"
)

func TestScaleCmd(t *testing.T) {
	test.Runs(t,
		// Ensure we don't segfault if user is not logged in
		test.ExecRun{
			Command: "convox scale",
			Env: map[string]string{
				"HOME":                  "/tmp/probablyNoConfigFileHere", // reset HOME to a location where there's not likely to be a convox config on the host
				"AWS_SECRET_ACCESS_KEY": "",
				"AWS_ACCESS_KEY_ID":     "",
			},
			Exit:   1,
			Stderr: "ERROR: couldn't initialize Rack client; please log in",
		},
	)
}
