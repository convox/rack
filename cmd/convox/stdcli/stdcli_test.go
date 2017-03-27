package stdcli_test

import (
	"os"
	"testing"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/urfave/cli.v1"
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

// TestCheckEnvVars ensures stdcli.CheckEnv() prints a warning if bool envvars aren't true/false/1/0
func TestCheckEnvVars(t *testing.T) {
	os.Setenv("RACK_PRIVATE", "foo")
	err := stdcli.CheckEnv()
	assert.Error(t, err)
	os.Unsetenv("RACK_PRIVATE")

	test.Runs(t,
		test.ExecRun{
			Command: "convox",
			Env:     map[string]string{"CONVOX_WAIT": "foo"},
			Exit:    1,
			Stderr:  "ERROR: 'foo' is not a valid value for environment variable CONVOX_WAIT (expected: [true false 1 0 ])\n",
		},
	)
}

func TestDebugEnv(t *testing.T) {
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

func TestDebugStdcli(t *testing.T) {
	oldDebug := os.Getenv("CONVOX_DEBUG")

	os.Setenv("CONVOX_DEBUG", "")
	d := stdcli.Debug()
	assert.Equal(t, d, false)

	os.Setenv("CONVOX_DEBUG", "true")
	d = stdcli.Debug()
	assert.Equal(t, d, true)

	os.Setenv("CONVOX_DEBUG", oldDebug)
}

func TestStdcliApp(t *testing.T) {
	app := stdcli.New()
	assert.Equal(t, "<command>", app.ArgsUsage)
	assert.Equal(t, "<command> [subcommand] [options...] [args...]", app.Usage)
	assert.Equal(t, "command-line application management", app.Description)
	assert.Equal(t, cli.BoolFlag{
		Name:   "help, h",
		Usage:  "show help",
		EnvVar: "",
		Hidden: false,
	}, cli.HelpFlag)
	args := []string{"convox foo"}
	err := app.Run(args)
	assert.NoError(t, err)

	stdcli.Spinner.Prefix = "Testing..."
	stdcli.Spinner.Start()
	stdcli.Spinner.Stop()
	assert.Equal(t, stdcli.Spinner.Prefix, "Testing...")
}
