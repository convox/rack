package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

func TestEventSend(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.EventSendOptions{
			Data: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Error:  options.String("error"),
			Status: options.String("status"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"action": "action",
				"data":   "k1=v1&k2=v2",
				"error":  "error",
				"status": "status",
			},
		}
		p.On("EventSend", "action", opts).Return(nil)
		err := c.Post("/events", ro, nil)
		require.NoError(t, err)
	})
}

func TestEventSendError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.EventSendOptions{
			Data: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Error:  options.String("error"),
			Status: options.String("status"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"action": "action",
				"data":   "k1=v1&k2=v2",
				"error":  "error",
				"status": "status",
			},
		}
		p.On("EventSend", "action", opts).Return(fmt.Errorf("err1"))
		err := c.Post("/events", ro, nil)
		require.EqualError(t, err, "err1")
	})
}
