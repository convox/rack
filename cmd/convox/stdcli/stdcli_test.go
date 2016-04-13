package stdcli_test

import (
	"testing"

	"github.com/convox/rack/cmd/convox/stdcli"
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
