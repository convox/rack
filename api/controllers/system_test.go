package controllers_test

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestSystemShow(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		system := &structs.System{
			Count:   3,
			Domain:  "foo",
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		p.On("SystemGet").Return(system, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"count\":3,\"domain\":\"foo\",\"image\":\"\",\"name\":\"test\",\"region\":\"us-test-1\",\"status\":\"running\",\"type\":\"t2.small\",\"version\":\"dev\"}")
		}
	})
}

func TestSystemShowRackFetchError(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		p.On("SystemGet").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}

func TestSystemUpdate(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		opts := structs.SystemUpdateOptions{
			InstanceCount: 5,
			InstanceType:  "t2.test",
			Version:       "latest",
		}

		p.On("SystemUpdate", opts).Return(nil)
		p.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "5")
		v.Add("type", "t2.test")
		v.Add("version", "latest")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"count\":3,\"domain\":\"\",\"image\":\"\",\"name\":\"test\",\"region\":\"us-test-1\",\"status\":\"running\",\"type\":\"t2.small\",\"version\":\"dev\"}")
		}
	})
}

func TestSystemUpdateAutoscaleCount(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		as := os.Getenv("AUTOSCALE")
		os.Setenv("AUTOSCALE", "true")
		defer os.Setenv("AUTOSCALE", as)

		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		p.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "5")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "scaling count prohibited when autoscale enabled")
		}
	})
}
func TestSystemUpdateBadCount(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		p.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "foo")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be numeric")
		}
	})

	Mock(func(p *structs.MockProvider) {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		p.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "-2")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be greater than 2")
		}
	})

	Mock(func(p *structs.MockProvider) {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		p.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "2")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be greater than 2")
		}
	})
}

func TestSystemCapacity(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		capacity := &structs.Capacity{
			ClusterCPU:     200,
			ClusterMemory:  2048,
			InstanceCPU:    100,
			InstanceMemory: 2048,
			ProcessCount:   10,
			ProcessMemory:  1928,
			ProcessCPU:     84,
			ProcessWidth:   3,
		}

		p.On("CapacityGet").Return(capacity, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system/capacity", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"cluster-cpu":200,"cluster-memory":2048,"instance-cpu":100,"instance-memory":2048,"process-count":10,"process-memory":1928,"process-cpu":84,"process-width":3}`)
		}
	})
}

func TestSystemCapacityError(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		p.On("CapacityGet").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system/capacity", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}
