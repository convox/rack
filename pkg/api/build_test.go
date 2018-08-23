package api_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildCreate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := structs.Build{Id: "build1"}
		b2 := structs.Build{}
		opts := structs.BuildCreateOptions{
			Description: options.String("description"),
			Manifest:    options.String("manifest"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"description": "description",
				"manifest":    "manifest",
				"url":         "https://host/path",
			},
		}
		p.On("BuildCreate", "app1", "https://host/path", opts).Return(&b1, nil)
		err := c.Post("/apps/app1/builds", ro, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildCreateNoCache(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := structs.Build{Id: "build1"}
		b2 := structs.Build{}
		opts := structs.BuildCreateOptions{
			Description: options.String("description"),
			Manifest:    options.String("manifest"),
			NoCache:     options.Bool(true),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"description": "description",
				"manifest":    "manifest",
				"no-cache":    "true",
				"url":         "https://host/path",
			},
		}
		p.On("BuildCreate", "app1", "https://host/path", opts).Return(&b1, nil)
		err := c.Post("/apps/app1/builds", ro, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildCreateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var b1 *structs.Build
		p.On("BuildCreate", "app1", "", structs.BuildCreateOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps/app1/builds", stdsdk.RequestOptions{}, b1)
		require.Nil(t, b1)
		require.EqualError(t, err, "err1")
	})
}

func TestBuildExport(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("BuildExport", "app1", "build1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			args.Get(2).(io.Writer).Write([]byte("data"))
		})
		res, err := c.GetStream("/apps/app1/builds/build1.tgz", stdsdk.RequestOptions{})
		require.NoError(t, err)
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "data", string(data))
	})
}

func TestBuildExportError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("BuildExport", "app1", "build1", mock.Anything).Return(fmt.Errorf("err1"))
		res, err := c.GetStream("/apps/app1/builds/build1.tgz", stdsdk.RequestOptions{})
		require.EqualError(t, err, "err1")
		require.Nil(t, res)
	})
}

func TestBuildGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := structs.Build{Id: "build1"}
		b2 := structs.Build{}
		p.On("BuildGet", "app1", "build1").Return(&b1, nil)
		err := c.Get("/apps/app1/builds/build1", stdsdk.RequestOptions{}, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var b1 *structs.App
		p.On("BuildGet", "app1", "build1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/builds/build1", stdsdk.RequestOptions{}, b1)
		require.Nil(t, b1)
		require.EqualError(t, err, "err1")
	})
}
