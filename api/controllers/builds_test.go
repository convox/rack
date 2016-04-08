package controllers

import (
	"encoding/json"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
)

func TestBuildDelete(t *testing.T) {
	// set current provider
	testProvider := &provider.TestProviderRunner{
		Build: structs.Build{
			Id: "build-id",
		},
	}
	provider.CurrentProvider = testProvider
	defer func() {
		//TODO: remove: as we arent updating all tests we need tos et current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	// setup expectations on current provider
	testProvider.On("BuildDelete", "app-name", "build-id").Return(&testProvider.Build, nil)

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
