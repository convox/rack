package controllers_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/client"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

func TestInstanceList(t *testing.T) {
	models.TestProvider = &provider.TestProvider{
		Instances: []structs.Instance{
			structs.Instance{},
			structs.Instance{},
			structs.Instance{},
		},
	}

	// setup expectations on current provider
	models.TestProvider.On("InstanceList").Return(models.TestProvider.Instances, nil)

	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")

	body := test.HTTPBody("GET", "http://convox/instances", nil)

	var resp []client.Instance

	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 3, len(resp))
	}
}

func TestInstanceTerminate(t *testing.T) {
	models.TestProvider.On("SystemGet").Return(nil, nil)

	os.Setenv("RACK", "convox-test")

	aws := test.StubAws(
		test.DeleteInstanceCycle("i-4a5513f4"),
	)
	defer aws.Close()

	body := test.HTTPBody("DELETE", "http://convox/instances/i-4a5513f4", nil)

	var resp map[string]bool
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, true, resp["success"])
	}
}
