package provider_test

import (
	"testing"
	"time"

	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/stretchr/testify/assert"
)

func TestReleaseLatestEmpty(t *testing.T) {
	p := provider.TestProvider{}

	p.Mock.On("ReleaseList", "myapp", int64(20)).Return(structs.Releases{}, nil)

	rs, err := p.ReleaseList("myapp", 20)

	assert.Nil(t, err)
	assert.Equal(t, (*structs.Release)(nil), rs.Latest())
}

func TestReleaseLatest(t *testing.T) {
	p := provider.TestProvider{}

	p.Mock.On("ReleaseList", "myapp", int64(20)).Return(structs.Releases{
		structs.Release{
			Id:       "RVFETUHHKKD",
			App:      "httpd",
			Build:    "BHINCLZYYVN",
			Env:      "foo=bar",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Created:  time.Unix(1459780542, 627770380).UTC(),
		},
		structs.Release{
			Id:       "RFVZFLKVTYO",
			App:      "httpd",
			Build:    "BNOARQMVHUO",
			Env:      "foo=bar",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Created:  time.Unix(1459709199, 166694813).UTC(),
		},
	}, nil)

	rs, err := p.ReleaseList("myapp", 20)

	assert.Nil(t, err)
	assert.Equal(t, &structs.Release{
		Id:       "RVFETUHHKKD",
		App:      "httpd",
		Build:    "BHINCLZYYVN",
		Env:      "foo=bar",
		Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
		Created:  time.Unix(1459780542, 627770380).UTC(),
	}, rs.Latest())
}
