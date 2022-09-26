package helpers_test

import (
	"bytes"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestStream(t *testing.T) {
	text := "hello world"
	w := &bytes.Buffer{}
	r := bytes.NewReader([]byte(text))
	err := helpers.Stream(w, r)
	require.NoError(t, err)
	require.Equal(t, text, w.String())
}
