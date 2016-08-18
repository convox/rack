package controllers_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestFormationList(t *testing.T) {
	formation := structs.Formation{
		structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}},
		structs.ProcessFormation{Name: "worker", Count: 3, CPU: 129, Memory: 1025, Ports: []int{4000, 4001}},
	}

	models.TestProvider = &provider.TestProvider{}
	models.TestProvider.On("FormationList", "myapp").Return(formation, nil)

	hf := test.NewHandlerFunc(controllers.HandlerFunc)

	if assert.Nil(t, hf.Request("GET", "/apps/myapp/formation", nil)) {
		hf.AssertCode(t, 200)
		hf.AssertJSON(t, `[
			{"balancer":"", "count":2, "cpu":128, "memory":1024, "name":"web", "ports":[3000,3001]},
			{"balancer":"", "count":3, "cpu":129, "memory":1025, "name":"worker", "ports":[4000,4001]}
		]`)
	}

	models.TestProvider.AssertExpectations(t)
}

func TestFormationListError(t *testing.T) {
	models.TestProvider = &provider.TestProvider{}
	models.TestProvider.On("FormationList", "myapp").Return(nil, fmt.Errorf("some error"))

	hf := test.NewHandlerFunc(controllers.HandlerFunc)
	hf.Request("GET", "/apps/myapp/formation", nil)

	hf.AssertCode(t, 500)
	hf.AssertError(t, "some error")

	models.TestProvider.AssertExpectations(t)
}
