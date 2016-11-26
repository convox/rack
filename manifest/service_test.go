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

func TestLabelsByPrefix(t *testing.T) {

	labels := manifest.Labels{
		"foofake": "label",
		"foo_foo": "under_bar",
		"foo-bar": "hypen-string",
		"te-st":   "hypen-string",
		"bahtest": "hypen-string",
	}

	s := manifest.Service{
		Labels: labels,
	}

	prefixed := s.LabelsByPrefix("foo")
	assert.Equal(t, map[string]string{
		"foofake": "label",
		"foo_foo": "under_bar",
		"foo-bar": "hypen-string",
	}, prefixed)
}

func TestNetworkName(t *testing.T) {
	networks := manifest.Networks{
		"foo": manifest.InternalNetwork{
			"external": manifest.ExternalNetwork{
				Name: "foonet",
			},
		},
	}

	s := manifest.Service{
		Networks: networks,
	}

	assert.Equal(t, s.NetworkName(), "foonet")
}

func TestDefaultNetworkName(t *testing.T) {
	networks := manifest.Networks{}

	s := manifest.Service{
		Networks: networks,
	}

	assert.Equal(t, s.NetworkName(), "")
}
