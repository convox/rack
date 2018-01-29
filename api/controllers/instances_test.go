package controllers_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

// func init() {
//   models.PauseNotifications = true
//   test.HandlerFunc = controllers.HandlerFunc
// }

func TestInstanceList(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		instances := structs.Instances{
			structs.Instance{
				Agent:     true,
				Cpu:       0.28,
				Id:        "test",
				Memory:    0.18,
				PrivateIp: "1.2.3.4",
				Processes: 5,
				PublicIp:  "2.3.4.5",
				Status:    "running",
				Started:   time.Unix(1475610360, 0).UTC(),
			},
		}

		p.On("InstanceList").Return(instances, nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("GET", "/instances", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertJSON(t, "[{\"agent\":true,\"cpu\":0.28,\"id\":\"test\",\"memory\":0.18,\"private-ip\":\"1.2.3.4\",\"processes\":5,\"public-ip\":\"2.3.4.5\",\"started\":\"2016-10-04T19:46:00Z\",\"status\":\"running\"}]")
		}
	})
}

func TestInstanceTerminate(t *testing.T) {
	Mock(func(p *structs.MockProvider) {
		p.On("InstanceTerminate", "i-1234").Return(nil)

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/instances/i-1234", nil)) {
			hf.AssertCode(t, 200)
			hf.AssertSuccess(t)
		}
	})

	Mock(func(p *structs.MockProvider) {
		p.On("InstanceTerminate", "i-1234").Return(fmt.Errorf("broken"))

		hf := test.NewHandlerFunc(controllers.HandlerFunc)

		if assert.Nil(t, hf.Request("DELETE", "/instances/i-1234", nil)) {
			hf.AssertCode(t, 500)
			hf.AssertError(t, "broken")
		}
	})
}
