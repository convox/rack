package api_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxApp = structs.App{
	Generation: "generation",
	Name:       "name",
	Release:    "release1",
	Sleep:      true,
	Status:     "created",
	Parameters: map[string]string{
		"p1": "v1",
		"p2": "v2",
	},
}

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
		a1 := fxApp
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
		a1 := fxApp
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
		a1 := fxApp
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

func TestAppList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := structs.Apps{fxApp, fxApp}
		a2 := structs.Apps{}
		p.On("AppList").Return(a1, nil)
		err := c.Get("/apps", stdsdk.RequestOptions{}, &a2)
		require.NoError(t, err)
		require.Equal(t, a1, a2)
	})
}

func TestAppListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var a1 structs.Apps
		p.On("AppList").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps", stdsdk.RequestOptions{}, &a1)
		require.EqualError(t, err, "err1")
		require.Nil(t, a1)
	})
}

func TestAppLogs(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		d1 := []byte("test")
		r1 := ioutil.NopCloser(bytes.NewReader(d1))
		opts := structs.LogsOptions{Since: options.Duration(2 * time.Minute)}
		p.On("AppLogs", "app1", opts).Return(r1, nil)
		r2, err := c.Websocket("/apps/app1/logs", stdsdk.RequestOptions{})
		require.NoError(t, err)
		d2, err := ioutil.ReadAll(r2)
		require.NoError(t, err)
		require.Equal(t, d1, d2)
	})
}

func TestAppLogsError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.LogsOptions{Since: options.Duration(2 * time.Minute)}
		p.On("AppLogs", "app1", opts).Return(nil, fmt.Errorf("err1"))
		r1, err := c.Websocket("/apps/app1/logs", stdsdk.RequestOptions{})
		require.NoError(t, err)
		require.NotNil(t, r1)
		d1, err := ioutil.ReadAll(r1)
		require.NoError(t, err)
		require.Equal(t, []byte("ERROR: err1\n"), d1)
	})
}

func TestAppUpdate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.AppUpdateOptions{
			Parameters: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			Sleep: options.Bool(true),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"parameters": "foo=bar&baz=qux",
				"sleep":      "true",
			},
		}
		p.On("AppUpdate", "app1", opts).Return(nil)
		err := c.Put("/apps/app1", ro, nil)
		require.NoError(t, err)
	})
}

func TestAppUpdateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppUpdate", "app1", structs.AppUpdateOptions{}).Return(fmt.Errorf("err1"))
		err := c.Put("/apps/app1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
