package stdcli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/stretchr/testify/assert"
)

func TestTableOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	old := stdcli.DefaultWriter
	stdcli.DefaultWriter.Stdout = buf
	defer func() {
		stdcli.DefaultWriter = old
	}()

	tb := stdcli.NewTable("FOO", "BAR")

	tb.AddRow("foo bar", "foo bar baz qux")
	tb.AddRow("bar foo baz", "foo")
	tb.Print()

	lines := strings.Split(buf.String(), "\n")

	assert.Equal(t, 4, len(lines))
	assert.Equal(t, "FOO          BAR", lines[0])
	assert.Equal(t, "foo bar      foo bar baz qux", lines[1])
	assert.Equal(t, "bar foo baz  foo", lines[2])
	assert.Equal(t, "", lines[3])
}
