package stdcli_test

import (
	"os"
	"testing"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestParseOptions(t *testing.T) {
	var opts map[string]string
	opts = stdcli.ParseOpts([]string{"--foo", "bar", "--key", "value"})
	assert.Equal(t, "bar", opts["foo"])
	assert.Equal(t, "value", opts["key"])

	opts = stdcli.ParseOpts([]string{"--foo=bar", "--key", "value"})
	assert.Equal(t, "bar", opts["foo"])
	assert.Equal(t, "value", opts["key"])

	opts = stdcli.ParseOpts([]string{"--foo=this", "is", "a bad idea"})
	assert.Equal(t, "this is a bad idea", opts["foo"])

	opts = stdcli.ParseOpts([]string{"--this", "--is=even", "worse"})
	assert.Equal(t, "even worse", opts["is"])
	_, ok := opts["this"]
	assert.Equal(t, true, ok)
}

func TestDebug(t *testing.T) {
	orig := os.Getenv("CONVOX_DEBUG")

	os.Setenv("CONVOX_DEBUG", "")
	assert.Equal(t, stdcli.Debug(), false)

	os.Setenv("CONVOX_DEBUG", "mraaaa")
	assert.Equal(t, stdcli.Debug(), true)

	os.Setenv("CONVOX_DEBUG", "true")
	assert.Equal(t, stdcli.Debug(), true)

	// restore original CONVOX_DEBUG value
	os.Setenv("CONVOX_DEBUG", orig)
}

// TestCheckEnvVars ensures stdcli.CheckEnv() prints a warning if bool envvars aren't true/false/1/0
func TestCheckEnvVars(t *testing.T) {
	os.Setenv("RACK_PRIVATE", "foo")

	err := stdcli.CheckEnv()
	assert.Error(t, err)

	test.Runs(t,
		test.ExecRun{
			Command: "convox",
			Env:     map[string]string{"CONVOX_WAIT": "foo"},
			Exit:    1,
			Stderr:  "ERROR: 'foo' is not a valid value for environment variable CONVOX_WAIT (expected: [true false 1 0 ])\n",
		},
	)
}
