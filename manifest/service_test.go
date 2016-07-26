package manifest_test

import (
	"testing"

	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

func TestTag(t *testing.T) {
	s := manifest.Service{
		Name: "foo",
	}
	assert.Equal(t, s.Tag("api"), "api/foo")

	s = manifest.Service{
		Name: "foo_bar",
	}
	assert.Equal(t, s.Tag("api"), "api/foo-bar")
}
