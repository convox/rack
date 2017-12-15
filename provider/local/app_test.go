package local_test

import (
	"testing"

	"github.com/convox/rack/structs"
	"github.com/stretchr/testify/assert"
)

func TestAppCreate(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	app, err := local.AppCreate("test", structs.AppCreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test", app.Name)
}

func TestAppCreateAlreadyExists(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	_, err = local.AppCreate("test", structs.AppCreateOptions{})
	assert.NoError(t, err)

	_, err = local.AppCreate("test", structs.AppCreateOptions{})
	assert.EqualError(t, err, "app already exists: test")
}

func TestAppDelete(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	_, err = local.AppCreate("test", structs.AppCreateOptions{})
	assert.NoError(t, err)

	_, err = local.AppGet("test")
	assert.NoError(t, err)

	err = local.AppDelete("test")
	assert.NoError(t, err)

	_, err = local.AppGet("test")
	assert.EqualError(t, err, "no such app: test")
}

func TestAppDeleteNonexistant(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	err = local.AppDelete("test")
	assert.EqualError(t, err, "no such app: test")
}

func TestAppGet(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	_, err = local.AppGet("test")
	assert.EqualError(t, err, "no such app: test")

	_, err = local.AppCreate("test", structs.AppCreateOptions{})
	assert.NoError(t, err)

	app, err := local.AppGet("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", app.Name)
}

func TestAppGetNonexistant(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	_, err = local.AppGet("testfoo")
	assert.EqualError(t, err, "no such app: testfoo")
}

func TestAppList(t *testing.T) {
	local, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(local)

	apps, err := local.AppList()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(apps))

	local.AppCreate("test1", structs.AppCreateOptions{})
	local.AppCreate("test3", structs.AppCreateOptions{})
	local.AppCreate("test2", structs.AppCreateOptions{})

	apps, err = local.AppList()
	assert.NoError(t, err)

	if assert.Equal(t, 3, len(apps)) {
		assert.Equal(t, "test1", apps[0].Name)
		assert.Equal(t, "test2", apps[1].Name)
		assert.Equal(t, "test3", apps[2].Name)
	}
}
