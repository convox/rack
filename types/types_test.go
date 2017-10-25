package types_test

import (
	"testing"

	"github.com/convox/rack/types"

	"github.com/stretchr/testify/assert"
)

func TestId(t *testing.T) {
	id := types.Id("A", 10)

	assert.Equal(t, "A", id[0:1])
	assert.Len(t, id, 10)
}

func TestIdUnique(t *testing.T) {
	id1 := types.Id("A", 10)
	id2 := types.Id("A", 10)

	assert.NotEqual(t, id1, id2)
}

func TestKey(t *testing.T) {
	key, err := types.Key(20)

	assert.NoError(t, err)
	assert.Len(t, key, 20)
}

func TestKeyTooLong(t *testing.T) {
	key, err := types.Key(100)

	assert.Error(t, err, "key too long")
	assert.Equal(t, "", key)
}

func TestKeyUnique(t *testing.T) {
	key1, err1 := types.Key(20)
	key2, err2 := types.Key(20)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, key1, key2)
}
