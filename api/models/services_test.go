package models

import (
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func TestCFParams(t *testing.T) {
	params := CFParams(map[string]string{
		"foo":                 "bar",
		"multi-az":            "true",
		"encrypted-storage":   "",
		"test-test-test-test": "TEST",
	})

	assert.Equal(t, "bar", params["Foo"])
	assert.Equal(t, "true", params["MultiAZ"])
	assert.Equal(t, "false", params["EncryptedStorage"])
	assert.Equal(t, "TEST", params["TestTestTestTest"])
}
