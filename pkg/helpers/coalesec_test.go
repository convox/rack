package helpers_test

import (
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestCoalesceInt(t *testing.T) {
	testData := []struct {
		given  []int
		expect int
	}{
		{
			given:  []int{1, 2, 3, 4, 5},
			expect: 1,
		},
		{
			given:  []int{1},
			expect: 1,
		},
		{
			given:  []int{0, 0, 0, 2, 4},
			expect: 2,
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expect, helpers.CoalesceInt(td.given...))
	}
}

func TestCoalesceString(t *testing.T) {
	testData := []struct {
		given  []string
		expect string
	}{
		{
			given:  []string{"1", "2"},
			expect: "1",
		},
		{
			given:  []string{"1"},
			expect: "1",
		},
		{
			given:  []string{"", "", "2", "3"},
			expect: "2",
		},
	}

	for _, td := range testData {
		require.Equal(t, td.expect, helpers.CoalesceString(td.given...))
	}
}
