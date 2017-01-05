package controllers_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	models.PauseNotifications = true
}

func TestServiceList(t *testing.T) {
	models.Test(t, func() {
		services := structs.Services{
			structs.Service{
				Name:       "memcached-1234",
				Stack:	    "-",
				Status:     "running",
				StatusReason: "",
				Type:       "memcached",

				Apps:	    nil,
				Exports:    map[string]string{"foo": "bar"},

				Outputs:    map[string]string{},
				Parameters: map[string]string{},
				Tags:       map[string]string{},
			},
		}

		models.TestProvider.On("ServiceList").Return(services, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/services", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}]")
		}
	})


	models.Test(t, func() {
		models.TestProvider.On("ServiceList").Return(nil, fmt.Errorf("unknown error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/services", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}

func TestServiceShow(t *testing.T) {
	/*
	models.Test(t, func() {
		models.TestProvider.On("ServiceGet").Return(nil, test.ErrorNotFound("no such service"))
		models.TestProvider.On("ServiceShow").Return(nil, test.ErrorNotFound("no such service"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/services/nonexistent-service-1234", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such service")
		}
	})
	*/

	models.Test(t, func() {
		services := structs.Services{
			structs.Service{
				Name:       "memcached-1234",
				Stack:	    "-",
				Status:     "running",
				StatusReason: "",
				Type:       "memcached",

				Apps:	    nil,
				Exports:    map[string]string{"foo": "bar"},

				Outputs:    map[string]string{},
				Parameters: map[string]string{},
				Tags:       map[string]string{},
			},
		}
		service := services[0]

		fmt.Println(service)
		//models.TestProvider.On("ServiceList").Return(services, nil)
		models.TestProvider.On("ServiceGet", "memcached-1234").Return(service, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/services/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}]")
		}
	})
}
