package helpers_test

import (
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestAtomStatus(t *testing.T) {
	testData := []struct {
		given, expect string
	}{
		{
			given:  "Failed",
			expect: "failed",
		},
		{
			given:  "Rollback",
			expect: "rollback",
		},
		{
			given:  "Deadline",
			expect: "updating",
		},
		{
			given:  "dfdf",
			expect: "running",
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expect, helpers.AtomStatus(td.given))
	}
}
