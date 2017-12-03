package controllers_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	models.PauseNotifications = true
}

func TestResourceList(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		resources := structs.Resources{
			structs.Resource{
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

		p.On("ResourceList").Return(resources, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/resources", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}]")
		}
	})

	Mock(func(p *provider.MockProvider) {
		p.On("ResourceList").Return(nil, fmt.Errorf("unknown error"))
		hf := test.NewHandlerFunc(controllers.HandlerFunc)
		if assert.Nil(t, hf.Request("GET", "/resources", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}

func TestResourceGet(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		p.On("ResourceGet", "nonexistent-resource-1234").Return(nil, test.ErrorNotFound("no such resource"))
		hf := test.NewHandlerFunc(controllers.HandlerFunc)
		if assert.Nil(t, hf.Request("GET", "/resources/nonexistent-resource-1234", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "no such resource")
		}
	})

	Mock(func(p *provider.MockProvider) {
		resource := structs.Resource{
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

		p.On("ResourceGet", "memcached-1234").Return(&resource, nil)
		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/resources/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}

func TestResourceCreate(t *testing.T) {

	v := url.Values{}
	v.Add("name", "memcached-1234")
	v.Add("type", "memcached")

	Mock(func(p *provider.MockProvider) {
		resource := structs.Resource{
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
		p.On("ResourceCreate", "memcached-1234", "memcached", map[string]string{}).Return(&resource, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)
		if assert.Nil(t, hf.Request("POST", "/resources", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}

func TestResourceDelete(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		resource := structs.Resource{
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
		p.On("ResourceGet", "memcached-1234").Return(&resource, nil)
		p.On("ResourceDelete", "memcached-1234").Return(&resource, nil)
		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/resources/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":null,\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}

// TestResourceShow ensures a resource can be shown
func TestResourceShow(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		services := structs.Resources{
			structs.Resource{
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
		p.On("ResourceGet", "memcached-1234").Return(&services[0], nil)
		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/resources/memcached-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"apps\":null,\"exports\":{\"foo\":\"bar\"},\"name\":\"memcached-1234\",\"status\":\"running\",\"status-reason\":\"\",\"type\":\"memcached\"}")
		}
	})
}
