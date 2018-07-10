package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

func TestAppCancel(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(&structs.App{Status: "updating"}, nil)
		p.On("AppCancel", "app1").Return(nil)
		err := c.Post("/apps/app1/cancel", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestAppCancelError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(&structs.App{Status: "updating"}, nil)
		p.On("AppCancel", "app1").Return(fmt.Errorf("err1"))
		err := c.Post("/apps/app1/cancel", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestAppCancelValidateNotUpdating(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(&structs.App{Status: "running"}, nil)
		err := c.Post("/apps/app1/cancel", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "app is not updating")
	})
}

func TestAppCancelValidateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps/app1/cancel", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestAppCreate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := structs.App{Name: "app1"}
		a2 := structs.App{}
		opts := structs.AppCreateOptions{
			Generation: options.String("2"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"name": "app1",
			},
		}
		p.On("AppCreate", "app1", opts).Return(&a1, nil)
		err := c.Post("/apps", ro, &a2)
		require.NoError(t, err)
		require.Equal(t, a1, a2)
	})
}

func TestAppCreateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.AppCreateOptions{
			Generation: options.String("2"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"name": "app1",
			},
		}
		p.On("AppCreate", "app1", opts).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps", ro, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestAppCreateGeneration1(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := structs.App{Name: "app1"}
		a2 := structs.App{}
		opts := structs.AppCreateOptions{
			Generation: options.String("1"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"generation": "1",
				"name":       "app1",
			},
		}
		p.On("AppCreate", "app1", opts).Return(&a1, nil)
		err := c.Post("/apps", ro, &a2)
		require.NoError(t, err)
		require.Equal(t, a1, a2)
	})
}
