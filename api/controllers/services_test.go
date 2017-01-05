package controllers_test

import (
	"fmt"
	"net/url"
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
				Name:         "memcached-1234",
				Stack:        "-",
				Status:       "running",
				StatusReason: "",
				Type:         "memcached",
				Apps:         nil,
				Exports:      map[string]string{"foo": "bar"},
				Outputs:      map[string]string{},
				Parameters:   map[string]string{},
				Tags:         map[string]string{},
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

func TestServiceGet(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("ServiceGet", "nonexistent-service-1234").Return(nil, test.ErrorNotFound("no such service"))
		hf := test.NewHandlerFunc(controllers.HandlerFunc)
		if assert.Nil(t, hf.Request("GET", "/services/nonexistent-service-1234", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "no such service")
		}
	})

	models.Test(t, func() {
		service := structs.Service{
			Name:         "memcached-1234",
			Stack:        "-",
			Status:       "running",
			StatusReason: "",
			Type:         "memcached",
			Apps:         nil,
			Exports:      map[string]string{"foo": "bar"},
			Outputs:      map[string]string{},
			Parameters:   map[string]string{},
			Tags:         map[string]string{},
		}

		models.TestProvider.On("ServiceGet", "memcached-1234").Return(&service, nil)
		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/services/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}

func TestServiceCreate(t *testing.T) {

	v := url.Values{}
	v.Add("name", "memcached-1234")
	v.Add("type", "memcached")

	models.Test(t, func() {
		service := structs.Service{
			Name:         "memcached-1234",
			Stack:        "-",
			Status:       "running",
			StatusReason: "",
			Type:         "memcached",
			Apps:         nil,
			Exports:      map[string]string{"foo": "bar"},
			Outputs:      map[string]string{},
			Parameters:   map[string]string{},
			Tags:         map[string]string{},
		}
		models.TestProvider.On("ServiceCreate", "memcached-1234", "memcached", map[string]string{}).Return(&service, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)
		if assert.Nil(t, hf.Request("POST", "/services", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}

func TestServiceDelete(t *testing.T) {
	models.Test(t, func() {
		service := structs.Service{
			Name:         "memcached-1234",
			Stack:        "-",
			Status:       "running",
			StatusReason: "",
			Type:         "memcached",
			Apps:         nil,
			Outputs:      map[string]string{},
			Parameters:   map[string]string{},
			Tags:         map[string]string{},
		}
		models.TestProvider.On("ServiceGet", "memcached-1234").Return(&service, nil)
		models.TestProvider.On("ServiceDelete", "memcached-1234").Return(&service, nil)
		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/services/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":null,\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
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
				Name:         "memcached-1234",
				Stack:        "-",
				Status:       "running",
				StatusReason: "",
				Type:         "memcached",
				Apps:         nil,
				Exports:      map[string]string{"foo": "bar"},
				Outputs:      map[string]string{},
				Parameters:   map[string]string{},
				Tags:         map[string]string{},
			},
		}
		models.TestProvider.On("ServiceGet", "memcached-1234").Return(&services[0], nil)
		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/services/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}
