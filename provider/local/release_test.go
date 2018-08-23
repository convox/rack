package local_test

import (
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseCreateGet(t *testing.T) {
	p, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(p)

	_, err = p.AppCreate("app", structs.AppCreateOptions{})
	assert.NoError(t, err)

	env := structs.Environment{
		"APP": "app",
		"FOO": "bar",
	}

	opts := structs.ReleaseCreateOptions{
		Env: options.String(env.String()),
	}
	rel, err := p.ReleaseCreate("app", opts)
	assert.NoError(t, err)

	if assert.NotNil(t, rel) {
		fetched, err := p.ReleaseGet("app", rel.Id)
		assert.NoError(t, err)

		assert.EqualValues(t, rel, fetched)
	}
}

func TestReleaseList(t *testing.T) {
	p, err := testProvider()
	assert.NoError(t, err)
	defer testProviderCleanup(p)

	_, err = p.AppCreate("app", structs.AppCreateOptions{})
	if !assert.NoError(t, err) {
		return
	}

	b1, err := p.BuildCreate("app", "", structs.BuildCreateOptions{})
	require.NoError(t, err)
	b2, err := p.BuildCreate("app", "", structs.BuildCreateOptions{})
	require.NoError(t, err)
	b3, err := p.BuildCreate("app", "", structs.BuildCreateOptions{})
	require.NoError(t, err)
	b4, err := p.BuildCreate("app", "", structs.BuildCreateOptions{})
	require.NoError(t, err)

	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String(b1.Id)})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Env: options.String(structs.Environment{"FOO": "bar"}.String())})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String(b2.Id)})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String(b3.Id)})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Env: options.String(structs.Environment{"FOO": "baz"}.String())})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String(b4.Id)})

	rs, err := p.ReleaseList("app", structs.ReleaseListOptions{})

	if assert.NoError(t, err) && assert.Len(t, rs, 6) {
		assert.Equal(t, b4.Id, rs[0].Build)
		assert.Equal(t, structs.Environment{"FOO": "baz"}.String(), rs[0].Env)
		assert.Equal(t, b3.Id, rs[1].Build)
		assert.Equal(t, structs.Environment{"FOO": "baz"}.String(), rs[1].Env)
		assert.Equal(t, b3.Id, rs[2].Build)
		assert.Equal(t, structs.Environment{"FOO": "bar"}.String(), rs[2].Env)
		assert.Equal(t, b2.Id, rs[3].Build)
		assert.Equal(t, structs.Environment{"FOO": "bar"}.String(), rs[3].Env)
		assert.Equal(t, b1.Id, rs[4].Build)
		assert.Equal(t, structs.Environment{"FOO": "bar"}.String(), rs[4].Env)
		assert.Equal(t, b1.Id, rs[5].Build)
		assert.Equal(t, structs.Environment{}.String(), rs[5].Env)
	}
}
