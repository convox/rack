package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTags(t *testing.T) {
	tests := []struct {
		in   string
		want map[string]string
	}{
		{"", map[string]string{}},
		{"A", map[string]string{}},
		{"A=b=c", map[string]string{"A": "b=c"}},
		{"A=1,A=2", map[string]string{"A": "1"}},
		{"A=1,B", map[string]string{"A": "1"}},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, parseTags(tt.in), tt.in)
	}
}

func TestTagsApplied(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		readback string
		rack     string
		version  string
		want     bool
	}{
		{"out of order", "Team=web,CostCenter=abc", "CostCenter=abc,Team=web", "System=convox,Type=rack", "", true},
		{"subset", "Team=web", "CostCenter=abc,Team=web", "", "", true},
		{"rack equal", "CostCenter=R", "", "System=convox,Type=rack,CostCenter=R", "", true},
		{"duplicate key", "A=1,A=2", "A=1", "", "", true},
		{"wrong value", "A=1,B=2", "A=1,B=3", "", "", false},
		{"missing key", "A=1", "", "A=9", "", false},
		{"present but different with rack match", "CostCenter=R", "CostCenter=X", "CostCenter=R", "", false},
		{"empty value", "A=", "", "", "", false},
		{"partial empty", "A=1,B=", "A=1", "", "", false},
		{"reserved in rack source", "Type=rack", "", "System=convox,Type=rack", "", false},
		{"below floor", "CostCenter=R", "", "CostCenter=R", "20260929232050", false},
		{"empty input", "", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := tt.version
			if version == "" {
				version = "21000101000000"
			}

			assert.Equal(t, tt.want, tagsApplied(tt.input, tt.readback, tt.rack, version))
		})
	}
}
