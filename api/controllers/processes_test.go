package controllers_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	models.PauseNotifications = true
}

func TestProcessGet(t *testing.T) {
	models.Test(t, func() {

		process := &structs.Process{
			ID:       "foo",
			App:      "myapp-staging",
			Group:    "group",
			Name:     "procname",
			Release:  "R123",
			Command:  "ls -la",
			Host:     "127.0.0.1",
			Image:    "image:tag",
			Instance: "i-1234",
			Ports:    []string{"80", "443"},
			CPU:      0.345,
			Memory:   0.456,
			Started:  time.Unix(1473483567, 0).UTC(),
		}

		models.TestProvider.On("ProcessGet", "myapp-staging", "foo").Return(process, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes/foo", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "{\"app\":\"myapp-staging\",\"command\":\"ls -la\",\"cpu\":0.345,\"group\":\"group\",\"host\":\"127.0.0.1\",\"id\":\"foo\",\"image\":\"image:tag\",\"instance\":\"i-1234\",\"memory\":0.456,\"name\":\"procname\",\"ports\":[\"80\",\"443\"],\"release\":\"R123\",\"started\":\"2016-09-10T04:59:27Z\"}")
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessGet", "myapp-staging", "blah").Return(nil, test.ErrorNotFound("no such process: blah"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes/blah", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process: blah")
		}
	})
}

func TestProcessList(t *testing.T) {
	models.Test(t, func() {
		processes := structs.Processes{
			structs.Process{
				ID:       "foo",
				App:      "myapp-staging",
				Group:    "group",
				Name:     "procname",
				Release:  "R123",
				Command:  "ls -la",
				Host:     "127.0.0.1",
				Image:    "image:tag",
				Instance: "i-1234",
				Ports:    []string{"80", "443"},
				CPU:      0.345,
				Memory:   0.456,
				Started:  time.Unix(1473483567, 0).UTC(),
			},
		}

		models.TestProvider.On("ProcessList", "myapp-staging").Return(processes, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"app\":\"myapp-staging\",\"command\":\"ls -la\",\"cpu\":0.345,\"group\":\"group\",\"host\":\"127.0.0.1\",\"id\":\"foo\",\"image\":\"image:tag\",\"instance\":\"i-1234\",\"memory\":0.456,\"name\":\"procname\",\"ports\":[\"80\",\"443\"],\"release\":\"R123\",\"started\":\"2016-09-10T04:59:27Z\"}]")
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessList", "myapp-staging").Return(nil, test.ErrorNotFound("no such process"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/apps/myapp-staging/processes", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process")
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessList", "myapp-staging").Return(nil, fmt.Errorf("unknown error"))

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
		Command: "test-command",
		Release: "R1234",
	}

	v := url.Values{}
	v.Add("command", "test-command")
	v.Add("release", "R1234")

	models.Test(t, func() {
		models.TestProvider.On("ProcessRun", "myapp-staging", "web", opts).Return("pid", nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes/web/run", v)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"success":true}`)
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessRun", "myapp-staging", "web", opts).Return("", test.ErrorNotFound("no such process"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes/web/run", v)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process")
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessRun", "myapp-staging", "web", opts).Return("", fmt.Errorf("unknown error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("POST", "/apps/myapp-staging/processes/web/run", v)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}

func TestProcessStop(t *testing.T) {
	models.Test(t, func() {
		models.TestProvider.On("ProcessStop", "myapp-staging", "p1234").Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp-staging/processes/p1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, `{"success":true}`)
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessStop", "myapp-staging", "p1234").Return(test.ErrorNotFound("no such process"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp-staging/processes/p1234", nil)) {
			hf.AssertCode(t, 404)
			hf.AssertError(t, "no such process")
		}
	})

	models.Test(t, func() {
		models.TestProvider.On("ProcessStop", "myapp-staging", "p1234").Return(fmt.Errorf("unknown error"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/apps/myapp-staging/processes/p1234", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "unknown error")
		}
	})
}
