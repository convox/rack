package controllers_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("PROVIDER", "test")
}

func TestBuildDelete(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		app := &structs.App{
			Name:    "myapp",
			Release: "R1234",
			Status:  "running",
		}
		release := &structs.Release{
			App:   "myapp",
			Build: "B1235",
			Id:    "R1234",
		}
		build := &structs.Build{
			App:         "myapp",
			Description: "desc",
			Ended:       time.Unix(1475611334, 0).UTC(),
			Id:          "B1234",
			Logs:        "",
			Manifest:    "",
			Reason:      "",
			Release:     "R2345",
			Started:     time.Unix(1475611334, 0).UTC(),
			Status:      "complete",
		}

		p.On("AppGet", "myapp").Return(app, nil)
		p.On("ReleaseGet", "myapp", "R1234").Return(release, nil)
		p.On("ReleaseDelete", "myapp", "B1234").Return(nil)
		p.On("BuildDelete", "myapp", "B1234").Return(build, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp/builds/B1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"app\":\"myapp\",\"description\":\"desc\",\"ended\":\"2016-10-04T20:02:14Z\",\"id\":\"B1234\",\"logs\":\"\",\"manifest\":\"\",\"reason\":\"\",\"release\":\"R2345\",\"started\":\"2016-10-04T20:02:14Z\",\"status\":\"complete\"}")
		}
	})
}

func TestBuildDeleteActive(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		app := &structs.App{
			Name:    "myapp",
			Release: "R1234",
			Status:  "running",
		}
		release := &structs.Release{
			App:   "myapp",
			Build: "B1234",
			Id:    "R1234",
		}

		p.On("AppGet", "myapp").Return(app, nil)
		p.On("ReleaseGet", "myapp", "R1234").Return(release, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp/builds/B1234", nil)) {
			hf.AssertCode(t, 400)
			hf.AssertJSON(t, "{\"error\":\"cannot delete build of active release: B1234\"}")
		}
	})
}
