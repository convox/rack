package models_test

import (
	"testing"

	"github.com/convox/rack/api/models"
	"github.com/stretchr/testify/assert"
)

// TestParseEnvLine ensures that simple comments, empty lines, and key value
// pairs, and invalid lines are correctly parsed by parseEnvLine
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

		{"heroku='likes to put things in single quotes'", true, "heroku", "likes to put things in single quotes"},

		{"K=V", true, "K", "V"},
		{"Key =value", true, "Key", "value"},
		{"KEY = 123", true, "KEY", "123"},
		{"k  =  292929", true, "k", "292929"},
	}

	for _, tr := range tests {
		k, v, err := models.ParseEnvLine(tr.line)
		if tr.valid {
			assert.NoError(t, err, "env line should be valid format")
		} else {
			assert.Error(t, err, "env line should be invalid format")
		}

		assert.Equal(t, tr.key, k, "for parsed env format key=value, invalid key returned")
		assert.Equal(t, tr.value, v, "for parsed env format key=value, invalid value returned")
	}
}
