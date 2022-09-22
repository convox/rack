package helpers_test

import (
	"testing"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestDuration(t *testing.T) {
	now := time.Now().UTC()
	testData := []struct {
		start, end time.Time
		expect     string
	}{
		{
			start:  now,
			end:    now,
			expect: "0s",
		},
		{
			start:  now,
			end:    now.Add(1*time.Hour + 31*time.Minute + 17*time.Second),
			expect: "91m17s",
		},
		{
			start:  now,
			end:    now.Add(-18 * time.Minute),
			expect: "0s",
		},
	}

	for _, td := range testData {
		d := helpers.Duration(td.start, td.end)
		require.Equal(t, td.expect, d)
	}
}
