package controllers_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

// func init() {
//   models.PauseNotifications = true
// }

func TestProcessGet(t *testing.T) {
	Mock(func(p *structs.MockProvider) {

		process := &structs.Process{
			Id:       "foo",
			App:      "myapp-staging",
			Name:     "procname",
			Release:  "R123",
			Command:  "ls -la",
			Host:     "127.0.0.1",
			Image:    "image:tag",
			Instance: "i-1234",
			Ports:    []string{"80", "443"},
			Cpu:      0.345,
			Memory:   0.456,
			Started:  time.Unix(1473483567, 0).UTC(),
		}

		p.On("ProcessGet", "myapp-staging", "foo").Return(process, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes/foo", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"app\":\"myapp-staging\",\"command\":\"ls -la\",\"cpu\":0.345,\"host\":\"127.0.0.1\",\"id\":\"foo\",\"image\":\"image:tag\",\"instance\":\"i-1234\",\"memory\":0.456,\"name\":\"procname\",\"ports\":[\"80\",\"443\"],\"release\":\"R123\",\"started\":\"2016-09-10T04:59:27Z\"}")
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessGet", "myapp-staging", "blah").Return(nil, test.ErrorNotFound("no such process: blah"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes/blah", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process: blah")
		}
	})
}

func TestProcessList(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		processes := structs.Processes{
			structs.Process{
				Id:       "foo",
				App:      "myapp-staging",
				Name:     "procname",
				Release:  "R123",
				Command:  "ls -la",
				Host:     "127.0.0.1",
				Image:    "image:tag",
				Instance: "i-1234",
				Ports:    []string{"80", "443"},
				Cpu:      0.345,
				Memory:   0.456,
				Started:  time.Unix(1473483567, 0).UTC(),
			},
		}

		p.On("ProcessList", "myapp-staging", structs.ProcessListOptions{}).Return(processes, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"app\":\"myapp-staging\",\"command\":\"ls -la\",\"cpu\":0.345,\"host\":\"127.0.0.1\",\"id\":\"foo\",\"image\":\"image:tag\",\"instance\":\"i-1234\",\"memory\":0.456,\"name\":\"procname\",\"ports\":[\"80\",\"443\"],\"release\":\"R123\",\"started\":\"2016-09-10T04:59:27Z\"}]")
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessList", "myapp-staging", structs.ProcessListOptions{}).Return(nil, test.ErrorNotFound("no such process"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process")
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessList", "myapp-staging", structs.ProcessListOptions{}).Return(nil, fmt.Errorf("unknown error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}

func TestProcessExecAttached(t *testing.T) {
}

func TestProcessRunAttached(t *testing.T) {
}

func TestProcessRunDetached(t *testing.T) {
	opts := structs.ProcessRunOptions{
		Command: options.String("test-command"),
		Release: options.String("R1234"),
		Service: options.String("web"),
	}

	v := url.Values{}
	v.Add("command", "test-command")
	v.Add("release", "R1234")

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessRun", "myapp-staging", opts).Return("pid", nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes/web/run", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"pid":"pid","success":true}`)
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessRun", "myapp-staging", opts).Return("", test.ErrorNotFound("no such process"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes/web/run", v)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process")
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessRun", "myapp-staging", opts).Return("", fmt.Errorf("unknown error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes/web/run", v)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}

func TestProcessStop(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		p.On("ProcessStop", "myapp-staging", "p1234").Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp-staging/processes/p1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"success":true}`)
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessStop", "myapp-staging", "p1234").Return(test.ErrorNotFound("no such process"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp-staging/processes/p1234", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process")
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("ProcessStop", "myapp-staging", "p1234").Return(fmt.Errorf("unknown error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp-staging/processes/p1234", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}
