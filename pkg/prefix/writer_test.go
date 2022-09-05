package prefix

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	var b bytes.Buffer
	w := NewWriter(&b, map[string]string{
		"test": "mock",
	})

	w.Write("test", bytes.NewReader([]byte("hello world")))

	require.Equal(t, "<mock>test</mock> | hello world\n", b.String())
}

func TestWritef(t *testing.T) {
	var b bytes.Buffer
	w := NewWriter(&b, map[string]string{
		"test": "mock",
	})

	w.Writef("test", "hello %s", "world")

	require.Equal(t, "<mock>test</mock> | hello world", b.String())
}
