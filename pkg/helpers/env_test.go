package helpers_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestEnvDiff(t *testing.T) {
	testData := []struct {
		env1   map[string]string
		env2   map[string]string
		expect string
	}{
		{
			env1: map[string]string{
				"a": "1",
				"b": "1",
			},
			env2: map[string]string{
				"a": "2",
				"c": "1",
			},
			expect: "add:c change:a remove:b",
		},
		{
			env1: map[string]string{
				"a": "1",
				"b": "1",
			},
			env2: map[string]string{
				"a": "1",
				"b": "1",
			},
			expect: "",
		},
	}

	envGen := func(m map[string]string) string {
		out := ""
		for k, v := range m {
			out = out + fmt.Sprintf("%s=%s\n", k, v)
		}
		return out
	}

	for _, td := range testData {
		got, err := helpers.EnvDiff(envGen(td.env1), envGen(td.env2))
		require.NoError(t, err)
		require.Equal(t, td.expect, got)
	}
}
