package controllers_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestFormationList(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		formation := structs.Formation{
			structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}},
			structs.ProcessFormation{Name: "worker", Count: 3, CPU: 129, Memory: 1025, Ports: []int{4000, 4001}},
		}

		p.On("FormationList", "myapp").Return(formation, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp/formation", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `[
				{"balancer":"", "count":2, "cpu":128, "memory":1024, "name":"web", "ports":[3000,3001]},
				{"balancer":"", "count":3, "cpu":129, "memory":1025, "name":"worker", "ports":[4000,4001]}
			]`)
		}
	})

	Mock(func(p *provider.MockProvider) {
		p.On("FormationList", "myapp").Return(nil, fmt.Errorf("some error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp/formation", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "some error")
		}
	})
}

func TestFormationSetAll(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 4, CPU: 200, Memory: 300, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")
		v.Add("cpu", "200")
		v.Add("memory", "300")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})
}

func TestFormationSetOne(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 200, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("cpu", "200")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})
}

func TestFormationSetFailedGet(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		p.On("FormationGet", "myapp", "web").Return(nil, fmt.Errorf("could not fetch"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")
		v.Add("cpu", "200")
		v.Add("memory", "300")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "could not fetch")
		}
	})
}

func TestFormationSetFailedSave(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 4, CPU: 200, Memory: 300, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(fmt.Errorf("could not save"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")
		v.Add("cpu", "200")
		v.Add("memory", "300")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "could not save")
		}
	})
}

func TestFormationSetEdgeCases(t *testing.T) {

	// count=-1 with older rack versions means no change
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 200, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)
		hf.SetVersion("20160602213112")

		v := url.Values{}
		v.Add("count", "-1")
		v.Add("cpu", "200")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})

	// count=-2 means no change
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 200, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "-2")
		v.Add("cpu", "200")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})

	// cpu=-1 means no change
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 4, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")
		v.Add("cpu", "-1")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})

	// memory=0 means no change
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 4, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")
		v.Add("memory", "0")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})

	// memory=-1 means no change
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}
		after := &structs.ProcessFormation{Name: "web", Count: 4, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)
		p.On("FormationSave", "myapp", after).Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "4")
		v.Add("memory", "-1")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})
}

func TestFormationSetNonNumeric(t *testing.T) {
	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("count", "foo")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "count must be numeric")
		}
	})

	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("cpu", "foo")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "cpu must be numeric")
		}
	})

	Mock(func(p *provider.MockProvider) {
		before := &structs.ProcessFormation{Name: "web", Count: 2, CPU: 128, Memory: 1024, Ports: []int{3000, 3001}}

		p.On("FormationGet", "myapp", "web").Return(before, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		v := url.Values{}
		v.Add("memory", "foo")

		if assert.Nil(t, hf.Request("POST", "/apps/myapp/formation/web", v)) {
			hf.AssertCode(t, 403)
			hf.AssertError(t, "memory must be numeric")
		}
	})
}
