package helpers_test

import (
	"strings"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestId(t *testing.T) {
	prefix := "test"
	length := 10
	id := helpers.Id(prefix, length)
	require.Equal(t, length, len(id))
	require.Equal(t, true, strings.HasPrefix(id, prefix))
}

func TestRandomString(t *testing.T) {
	length := 10
	s, err := helpers.RandomString(length)
	require.NoError(t, err)
	require.Equal(t, length, len(s))
}
