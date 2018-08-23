package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
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
		var a1 *structs.App
		opts := structs.AppCreateOptions{
			Generation: options.String("2"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"name": "app1",
			},
		}
		p.On("AppCreate", "app1", opts).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps", ro, a1)
		require.Nil(t, a1)
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

func TestAppDelete(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppDelete", "app1").Return(nil)
		err := c.Delete("/apps/app1", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestAppDeleteError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppDelete", "app1").Return(fmt.Errorf("err1"))
		err := c.Delete("/apps/app1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestAppGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := structs.App{Name: "app1"}
		a2 := structs.App{}
		p.On("AppGet", "app1").Return(&a1, nil)
		err := c.Get("/apps/app1", stdsdk.RequestOptions{}, &a2)
		require.NoError(t, err)
		require.Equal(t, a1, a2)
	})
}

func TestAppGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var a1 *structs.App
		p.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1", stdsdk.RequestOptions{}, a1)
		require.Nil(t, a1)
		require.EqualError(t, err, "err1")
	})
}
