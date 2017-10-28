package controllers_test

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestSystemShow(t *testing.T) {
	models.Test(t, func() {
		system := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		models.TestProvider.On("SystemGet").Return(system, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"count":3,"name":"test","region":"us-test-1","status":"running","type":"t2.small","version":"dev"}`)
		}
	})
}

func TestSystemShowRackFetchError(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("SystemGet").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}

func TestSystemUpdate(t *testing.T) {
	models.Test(t, func() {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}
		change := structs.System{
			Count:   5,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.test",
			Version: "latest",
		}

		models.TestProvider.On("SystemGet").Return(before, nil)
		models.TestProvider.On("SystemSave", change).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "5")
		v.Add("type", "t2.test")
		v.Add("version", "latest")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"count":5,"name":"test","region":"us-test-1","status":"running","type":"t2.test","version":"latest"}`)
		}
	})
}

func TestSystemUpdateRackFetchError(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("SystemGet").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("PUT", "/system", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}

func TestSystemUpdateCountNoChange(t *testing.T) {
	models.Test(t, func() {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}
		change := structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "latest",
		}

		models.TestProvider.On("SystemGet").Return(before, nil)
		models.TestProvider.On("SystemSave", change).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "-1")
		v.Add("version", "latest")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"count":3,"name":"test","region":"us-test-1","status":"running","type":"t2.small","version":"latest"}`)
		}
	})
}

func TestSystemUpdateAutoscaleCount(t *testing.T) {
	models.Test(t, func() {
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

		models.TestProvider.On("SystemGet").Return(before, nil)

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
	models.Test(t, func() {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		models.TestProvider.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "foo")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be numeric")
		}
	})

	models.Test(t, func() {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		models.TestProvider.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "-2")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be greater than 2")
		}
	})

	models.Test(t, func() {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		models.TestProvider.On("SystemGet").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "2")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be greater than 2")
		}
	})
}

func TestSystemUpdateSaveError(t *testing.T) {
	models.Test(t, func() {
		before := &structs.System{
			Count:   3,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}
		change := structs.System{
			Count:   4,
			Name:    "test",
			Region:  "us-test-1",
			Status:  "running",
			Type:    "t2.small",
			Version: "dev",
		}

		models.TestProvider.On("SystemGet").Return(before, nil)
		models.TestProvider.On("SystemSave", change).Return(fmt.Errorf("bad save"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")

		if assert.Nil(t, hf.Request("PUT", "/system", v)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "bad save")
		}
	})
}

func TestSystemCapacity(t *testing.T) {
	models.Test(t, func() {
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

		models.TestProvider.On("CapacityGet").Return(capacity, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system/capacity", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"cluster-cpu":200,"cluster-memory":2048,"instance-cpu":100,"instance-memory":2048,"process-count":10,"process-memory":1928,"process-cpu":84,"process-width":3}`)
		}
	})
}

func TestSystemCapacityError(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("CapacityGet").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system/capacity", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}

func TestSystemReleases(t *testing.T) {
	models.Test(t, func() {
		releases := structs.Releases{
			structs.Release{Id: "R0000001", App: "test", Build: "B0000001", Created: time.Date(2016, 3, 4, 5, 6, 7, 12, time.UTC)},
			structs.Release{Id: "R0000002", App: "test", Build: "B0000002", Created: time.Date(2016, 3, 4, 9, 6, 7, 14, time.UTC)},
		}

		models.TestProvider.On("SystemReleases").Return(releases, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system/releases", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `[
				{"app":"test","build":"B0000001","created":"2016-03-04T05:06:07.000000012Z","env":"","id":"R0000001","manifest":""},
				{"app":"test","build":"B0000002","created":"2016-03-04T09:06:07.000000014Z","env":"","id":"R0000002","manifest":""}
			]`)
		}
	})
}

func TestSystemReleasesError(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("SystemReleases").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/system/releases", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}
