package controllers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("PROVIDER", "test")
	test.HandlerFunc = HandlerFunc
}

func TestBuildDelete(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		App: structs.App{
			Name:    "app-name",
			Release: "release-id",
		},
		Build: structs.Build{
			Id: "build-id",
		},
		Release: structs.Release{
			Id:    "release-id",
			Build: "not-build-id",
		},
	}

	// setup expectations on current provider
	models.TestProvider.On("AppGet", "app-name").Return(&models.TestProvider.App, nil)
	models.TestProvider.On("BuildGet", "app-name", "build-id").Return(&models.TestProvider.Build, nil)
	models.TestProvider.On("ReleaseGet", "app-name", "release-id").Return(&models.TestProvider.Release, nil)
	models.TestProvider.On("BuildDelete", "app-name", "build-id").Return(&models.TestProvider.Build, nil)
	models.TestProvider.On("ReleaseDelete", "app-name", "build-id").Return(nil)

	// make request
	body := test.HTTPBody("DELETE", "http://convox/apps/app-name/builds/build-id", nil)

	// assert on expectations
	models.TestProvider.AssertExpectations(t)

	// assert on response
	resp := new(structs.Build)
	err := json.Unmarshal([]byte(body), resp)
	if assert.Nil(t, err) {
		assert.Equal(t, "build-id", resp.Id)
	}
}

func TestBuildDeleteActive(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		App: structs.App{
			Name:    "httpd",
			Release: "release-id",
		},
		Build: structs.Build{
			Id: "BHINCLZYYVN",
		},
		Release: structs.Release{
			Id:    "release-id",
			Build: "BHINCLZYYVN",
		},
	}

	models.TestProvider.On("AppGet", "httpd").Return(&models.TestProvider.App, nil)
	models.TestProvider.On("BuildGet", "httpd", "BHINCLZYYVN").Return(&models.TestProvider.Build, nil)
	models.TestProvider.On("ReleaseGet", "httpd", "release-id").Return(&models.TestProvider.Release, nil)

	body := test.HTTPBody("DELETE", "http://convox/apps/httpd/builds/BHINCLZYYVN", nil)

	// assert on expectations
	models.TestProvider.AssertExpectations(t)

	// assert on response
	resp := make(map[string]string)
	err := json.Unmarshal([]byte(body), &resp)
	if assert.Nil(t, err) {
		fmt.Fprintf(os.Stderr, "%s\n", resp)
		assert.Equal(t, "cannot delete build contained in active release", resp["error"])
	}
}
