package controllers_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func init() {
	models.PauseNotifications = true
	test.HandlerFunc = controllers.HandlerFunc
}

func TestInstanceList(t *testing.T) {
	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test-cluster")

	aws := test.StubAws(
		test.DescribeConvoxStackCycle("convox-test"),
		test.ListContainerInstancesCycle("convox-test-cluster"),
		test.DescribeContainerInstancesCycle("convox-test-cluster"),
		test.DescribeInstancesCycle(),
	)
	defer aws.Close()

	body := test.HTTPBody("GET", "http://convox/instances", nil)

	var resp []client.Instance
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, 3, len(resp))
	}
}
