package controllers_test

import (
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("PROVIDER", "test")
}

func TestBuildDelete(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("ReleaseDelete", "example", "B1234").Return(nil)
		models.TestProvider.On("BuildDelete", "example", "B1234").Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/example/builds/B1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"app\":\"\",\"description\":\"\",\"ended\":\"0001-01-01T00:00:00Z\",\"id\":\"\",\"logs\":\"\",\"manifest\":\"\",\"reason\":\"\",\"release\":\"\",\"started\":\"0001-01-01T00:00:00Z\",\"status\":\"\"}")
		}
	})
}
