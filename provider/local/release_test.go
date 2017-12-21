package local_test

import (
	"testing"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/stretchr/testify/assert"
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
		Build: options.String("BTEST"),
		Env:   options.String(env.String()),
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

	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String("B1")})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Env: options.String(structs.Environment{"FOO": "bar"}.String())})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String("B2")})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String("B3")})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Env: options.String(structs.Environment{"FOO": "baz"}.String())})
	p.ReleaseCreate("app", structs.ReleaseCreateOptions{Build: options.String("B4")})

	rs, err := p.ReleaseList("app", structs.ReleaseListOptions{})

	if assert.NoError(t, err) && assert.Len(t, rs, 6) {
		assert.Equal(t, "B4", rs[0].Build)
		assert.Equal(t, structs.Environment{"FOO": "baz"}.String(), rs[0].Env)
		assert.Equal(t, "B3", rs[1].Build)
		assert.Equal(t, structs.Environment{"FOO": "baz"}.String(), rs[1].Env)
		assert.Equal(t, "B3", rs[2].Build)
		assert.Equal(t, structs.Environment{"FOO": "bar"}.String(), rs[2].Env)
		assert.Equal(t, "B2", rs[3].Build)
		assert.Equal(t, structs.Environment{"FOO": "bar"}.String(), rs[3].Env)
		assert.Equal(t, "B1", rs[4].Build)
		assert.Equal(t, structs.Environment{"FOO": "bar"}.String(), rs[4].Env)
		assert.Equal(t, "B1", rs[5].Build)
		assert.Equal(t, structs.Environment{}.String(), rs[5].Env)
	}
}
