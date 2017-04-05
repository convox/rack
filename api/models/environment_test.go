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
		key   string
		value string
	}{
		{"", "", ""},
		{" ", "", ""},
		{"	 ", "", ""},

		{"#", "", ""},
		{"# ", "", ""},
		{"  #", "", ""},
		{"	 #", "", ""},
		{"# comment", "", ""},

		{"An Invalid line", "", ""},

		{"heroku='likes to put things in single quotes'", "heroku", "likes to put things in single quotes"},

		{"K=V", "K", "V"},
		{"Key =value", "Key", "value"},
		{"KEY = 123", "KEY", "123"},
		{"k  =  292929", "k", "292929"},
	}

	for _, tr := range tests {
		k, v := models.ParseEnvLine(tr.line)

		assert.Equal(t, tr.key, k, "for parsed env format key=value, invalid key returned")
		assert.Equal(t, tr.value, v, "for parsed env format key=value, invalid value returned")
	}
}
