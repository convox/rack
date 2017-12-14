package local_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/convox/rack/structs"
	"github.com/stretchr/testify/assert"
)

func TestObjectStoreFetch(t *testing.T) {
	p, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(p)

	_, err = p.AppCreate("app")
	assert.NoError(t, err)

	data := bytes.NewBuffer([]byte("object to store"))
	expect := &structs.Object{Key: "mykey"}

	obj, err := p.ObjectStore("app", "mykey", data, structs.ObjectStoreOptions{})
	assert.NoError(t, err)
	assert.Equal(t, expect, obj)

	fetched, err := p.ObjectFetch("app", "mykey")
	assert.NoError(t, err)

	read, err := ioutil.ReadAll(fetched)
	assert.NoError(t, err)

	assert.Equal(t, "object to store", string(read))
}
