package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		line  string
		valid bool
		key   string
		value string
	}{
		{"", true, "", ""},
		{" ", true, "", ""},
		{"	 ", true, "", ""},

		{"#", true, "", ""},
		{"# ", true, "", ""},
		{"  #", true, "", ""},
		{"	 #", true, "", ""},
		{"# comment", true, "", ""},

		{"An Invalid line", false, "", ""},

		{"K=V", true, "K", "V"},
		{"Key =value", true, "Key", "value"},
		{"KEY = 123", true, "KEY", "123"},
		{"k  =  292929", true, "k", "292929"},
	}

	for _, test := range tests {
		k, v, err := parseEnvLine(test.line)
		if test.valid {
			assert.NoError(t, err, "env line should be valid format")
		} else {
			assert.Error(t, err, "env line should be invalid format")
		}

		assert.Equal(t, test.key, k, "for parsed env format key=value, invalid key resturned")
		assert.Equal(t, test.value, v, "for parsed env format key=value, invalid value resturned")
	}
}
