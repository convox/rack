package api_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fxBuild = structs.Build{
	Id:          "build1",
	App:         "app1",
	Description: "description",
	Logs:        "logs",
	Manifest:    "manifest",
	Process:     "process",
	Release:     "release1",
	Reason:      "reason",
	Status:      "status",
	Started:     time.Now().UTC(),
	Ended:       time.Now().UTC(),
}

func TestBuildCreate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := fxBuild
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
		b1 := fxBuild
		var b2 structs.Build
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
		b1 := fxBuild
		b2 := structs.Build{}
		p.On("BuildGet", "app1", "build1").Return(&b1, nil)
		err := c.Get("/apps/app1/builds/build1", stdsdk.RequestOptions{}, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var b1 *structs.Build
		p.On("BuildGet", "app1", "build1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/builds/build1", stdsdk.RequestOptions{}, b1)
		require.Nil(t, b1)
		require.EqualError(t, err, "err1")
	})
}

func TestBuildImport(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := fxBuild
		b2 := structs.Build{}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("data"),
		}
		p.On("BuildImport", "app1", mock.Anything).Return(&b1, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "data", string(data))
		})
		err := c.Post("/apps/app1/builds/import", ro, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildImportError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var b1 *structs.Build
		p.On("BuildImport", "app1", mock.Anything).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps/app1/builds/import", stdsdk.RequestOptions{}, b1)
		require.EqualError(t, err, "err1")
		require.Nil(t, b1)
	})
}

func TestBuildLogs(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		d1 := []byte("test")
		r1 := ioutil.NopCloser(bytes.NewReader(d1))
		opts := structs.LogsOptions{Since: options.Duration(2 * time.Minute)}
		p.On("BuildLogs", "app1", "build1", opts).Return(r1, nil)
		r2, err := c.Websocket("/apps/app1/builds/build1/logs", stdsdk.RequestOptions{})
		require.NoError(t, err)
		d2, err := ioutil.ReadAll(r2)
		require.NoError(t, err)
		require.Equal(t, d1, d2)
	})
}

func TestBuildLogsError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.LogsOptions{Since: options.Duration(2 * time.Minute)}
		p.On("BuildLogs", "app1", "build1", opts).Return(nil, fmt.Errorf("err1"))
		r1, err := c.Websocket("/apps/app1/builds/build1/logs", stdsdk.RequestOptions{})
		require.NoError(t, err)
		require.NotNil(t, r1)
		d1, err := ioutil.ReadAll(r1)
		require.NoError(t, err)
		require.Equal(t, []byte("ERROR: err1\n"), d1)
	})
}

func TestBuildList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := structs.Builds{fxBuild, fxBuild}
		b2 := structs.Builds{}
		opts := structs.BuildListOptions{
			Limit: options.Int(10),
		}
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"limit": "10",
			},
		}
		p.On("BuildList", "app1", opts).Return(b1, nil)
		err := c.Get("/apps/app1/builds", ro, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var b1 structs.Builds
		p.On("BuildList", "app1", structs.BuildListOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/builds", stdsdk.RequestOptions{}, &b1)
		require.EqualError(t, err, "err1")
		require.Nil(t, b1)
	})
}

func TestBuildUpdate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		b1 := fxBuild
		b2 := structs.Build{}
		opts := structs.BuildUpdateOptions{
			Ended:    options.Time(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)),
			Logs:     options.String("logs"),
			Manifest: options.String("manifest"),
			Release:  options.String("release1"),
			Started:  options.Time(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)),
			Status:   options.String("status"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"ended":    "20180101.000000.000000000",
				"logs":     "logs",
				"manifest": "manifest",
				"release":  "release1",
				"started":  "20180101.000000.000000000",
				"status":   "status",
			},
		}
		p.On("BuildUpdate", "app1", "build1", opts).Return(&b1, nil)
		err := c.Put("/apps/app1/builds/build1", ro, &b2)
		require.NoError(t, err)
		require.Equal(t, b1, b2)
	})
}

func TestBuildUpdateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var b1 *structs.Build
		p.On("BuildUpdate", "app1", "build1", structs.BuildUpdateOptions{}).Return(b1, fmt.Errorf("err1"))
		err := c.Put("/apps/app1/builds/build1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
