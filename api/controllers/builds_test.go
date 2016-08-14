package controllers

import (
	"encoding/json"
	"testing"

	"github.com/convox/rack/provider"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestBuildDelete(t *testing.T) {
	// set current provider
	testProvider := &provider.TestProviderRunner{
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
	provider.CurrentProvider = testProvider
	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	// setup expectations on current provider
	testProvider.On("AppGet", "app-name").Return(&testProvider.App, nil)
	testProvider.On("BuildGet", "app-name", "build-id").Return(&testProvider.Build, nil)
	testProvider.On("ReleaseGet", "app-name", "release-id").Return(&testProvider.Release, nil)
	testProvider.On("BuildDelete", "app-name", "build-id").Return(&testProvider.Build, nil)
	testProvider.On("ReleaseDelete", "app-name", "build-id").Return(nil)

	// make request
	body := test.HTTPBody("DELETE", "http://convox/apps/app-name/builds/build-id", nil)

	// assert on expectations
	testProvider.AssertExpectations(t)

	// assert on response
	resp := new(structs.Build)
	err := json.Unmarshal([]byte(body), resp)
	if assert.Nil(t, err) {
		assert.Equal(t, "build-id", resp.Id)
	}
}
