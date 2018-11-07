package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxRelease = structs.Release{
	Id:       "release1",
	App:      "app1",
	Build:    "build1",
	Env:      "env",
	Manifest: "manifest",
	Created:  time.Now().UTC(),
}

func TestReleaseCreate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxRelease
		r2 := structs.Release{}
		opts := structs.ReleaseCreateOptions{
			Build: options.String("build1"),
			Env:   options.String("env"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"build": "build1",
				"env":   "env",
			},
		}
		p.On("ReleaseCreate", "app1", opts).Return(&r1, nil)
		err := c.Post("/apps/app1/releases", ro, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestReleaseCreateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Release
		opts := structs.ReleaseCreateOptions{
			Build: options.String("build1"),
			Env:   options.String("env"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"build": "build1",
				"env":   "env",
			},
		}
		p.On("ReleaseCreate", "app1", opts).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps/app1/releases", ro, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestReleaseGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxRelease
		r2 := structs.Release{}
		p.On("ReleaseGet", "app1", "release1").Return(&r1, nil)
		err := c.Get("/apps/app1/releases/release1", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestReleaseGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Release
		p.On("ReleaseGet", "app1", "release1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/releases/release1", stdsdk.RequestOptions{}, r1)
		require.Nil(t, r1)
		require.EqualError(t, err, "err1")
	})
}

func TestReleaseList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := structs.Releases{fxRelease, fxRelease}
		r2 := structs.Releases{}
		opts := structs.ReleaseListOptions{
			Limit: options.Int(1),
		}
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"limit": "1",
			},
		}
		p.On("ReleaseList", "app1", opts).Return(r1, nil)
		err := c.Get("/apps/app1/releases", ro, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestReleaseListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 structs.Releases
		p.On("ReleaseList", "app1", structs.ReleaseListOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/releases", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestReleasePromote(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(&structs.App{Status: "running"}, nil)
		opts := structs.ReleasePromoteOptions{
			Min: options.Int(1),
			Max: options.Int(2),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"min": "1",
				"max": "2",
			},
		}
		p.On("ReleasePromote", "app1", "release1", opts).Return(nil)
		err := c.Post("/apps/app1/releases/release1/promote", ro, nil)
		require.NoError(t, err)
	})
}

func TestReleasePromoteError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(&structs.App{Status: "running"}, nil)
		p.On("ReleasePromote", "app1", "release1", structs.ReleasePromoteOptions{}).Return(fmt.Errorf("err1"))
		err := c.Post("/apps/app1/releases/release1/promote", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestReleasePromoteNotRunning(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(&structs.App{Status: "other"}, nil)
		p.On("ReleasePromote", "app1", "release1").Return(nil)
		err := c.Post("/apps/app1/releases/release1/promote", stdsdk.RequestOptions{}, nil)
		require.Error(t, err, "app is currently updating")
	})
}
