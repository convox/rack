package stdcli_test

import (
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/cmd/convox/stdcli"
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
}
