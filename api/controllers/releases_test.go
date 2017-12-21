package controllers_test

import (
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestReleaseList(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		releases := structs.Releases{
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
		}

		p.On("ReleaseList", "example", structs.ReleaseListOptions{}).Return(releases, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/example/releases", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"app\":\"httpd\",\"build\":\"BHINCLZYYVN\",\"created\":\"2016-04-04T14:35:42.62777038Z\",\"env\":\"foo=bar\",\"id\":\"RVFETUHHKKD\",\"manifest\":\"web:\\n  image: httpd\\n  ports:\\n  - 80:80\\n\",\"status\":\"\"},{\"app\":\"httpd\",\"build\":\"BNOARQMVHUO\",\"created\":\"2016-04-03T18:46:39.166694813Z\",\"env\":\"foo=bar\",\"id\":\"RFVZFLKVTYO\",\"manifest\":\"web:\\n  image: httpd\\n  ports:\\n  - 80:80\\n\",\"status\":\"\"}]")
		}
	})
}
