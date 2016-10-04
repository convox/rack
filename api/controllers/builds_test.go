package controllers_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("PROVIDER", "test")
}

func TestBuildDelete(t *testing.T) {
	models.Test(t, func() {
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

		models.TestProvider.On("ReleaseDelete", "example", "B1234").Return(nil)
		models.TestProvider.On("BuildDelete", "example", "B1234").Return(build, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/example/builds/B1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"app\":\"myapp\",\"description\":\"desc\",\"ended\":\"2016-10-04T20:02:14Z\",\"id\":\"B1234\",\"logs\":\"\",\"manifest\":\"\",\"reason\":\"\",\"release\":\"R2345\",\"started\":\"2016-10-04T20:02:14Z\",\"status\":\"complete\"}")
		}
	})
}
