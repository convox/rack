package helpers_test

import (
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestFormatYAML(t *testing.T) {
	sample := `
key1:
    key2: val1
    key3:
        - key4:
              key5: val

---
key1: 345
`
	expect := "key1:\n  key2: val1\n  key3:\n  - key4:\n      key5: val\n---\nkey1: 345\n"
	b, err := helpers.FormatYAML([]byte(sample))
	require.NoError(t, err)
	require.Equal(t, expect, string(b))
}
