package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxResource = structs.Resource{
	Name:       "resource1",
	Parameters: map[string]string{"k1": "v1", "k2": "v2"},
	Status:     "status",
	Type:       "type",
	Url:        "https://example.org/path",
	Apps:       structs.Apps{fxApp, fxApp},
}

var fxResourceType = structs.ResourceType{
	Name: "name",
	Parameters: structs.ResourceParameters{
		{Default: "default1", Description: "description1", Name: "name1"},
		{Default: "default2", Description: "description2", Name: "name2"},
	},
}

func TestResourceGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxResource
		r2 := structs.Resource{}
		p.On("ResourceGet", "app1", "resource1").Return(&r1, nil)
		err := c.Get("/apps/app1/resources/resource1", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestResourceGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Resource
		p.On("ResourceGet", "app1", "resource1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/resources/resource1", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestResourceList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := structs.Resources{fxResource, fxResource}
		r2 := structs.Resources{}
		p.On("ResourceList", "app1").Return(r1, nil)
		err := c.Get("/apps/app1/resources", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestResourceListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 structs.Resources
		p.On("ResourceList", "app1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/resources", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceCreate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxResource
		r2 := structs.Resource{}
		opts := structs.ResourceCreateOptions{
			Name:       options.String("resource1"),
			Parameters: map[string]string{"k1": "v1", "k2": "v2"},
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"name":       "resource1",
				"kind":       "type",
				"parameters": "k1=v1&k2=v2",
			},
		}
		p.On("SystemResourceCreate", "type", opts).Return(&r1, nil)
		err := c.Post("/resources", ro, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceCreateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Resource
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"kind": "type",
			},
		}
		p.On("SystemResourceCreate", "type", structs.ResourceCreateOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/resources", ro, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceDelete(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("SystemResourceDelete", "resource1").Return(nil)
		err := c.Delete("/resources/resource1", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestSystemResourceDeleteError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("SystemResourceDelete", "resource1").Return(fmt.Errorf("err1"))
		err := c.Delete("/resources/resource1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestSystemResourceGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxResource
		r2 := structs.Resource{}
		p.On("SystemResourceGet", "resource1").Return(&r1, nil)
		err := c.Get("/resources/resource1", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Resource
		p.On("SystemResourceGet", "resource1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/resources/resource1", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceLink(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxResource
		r2 := structs.Resource{}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"app": "app1",
			},
		}
		p.On("SystemResourceLink", "resource1", "app1").Return(&r1, nil)
		err := c.Post("/resources/resource1/links", ro, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceLinkError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Resource
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"app": "app1",
			},
		}
		p.On("SystemResourceLink", "resource1", "app1").Return(nil, fmt.Errorf("err1"))
		err := c.Post("/resources/resource1/links", ro, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := structs.Resources{fxResource, fxResource}
		r2 := structs.Resources{}
		p.On("SystemResourceList").Return(r1, nil)
		err := c.Get("/resources", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 structs.Resources
		p.On("SystemResourceList").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/resources", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceTypes(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := structs.ResourceTypes{fxResourceType, fxResourceType}
		r2 := structs.ResourceTypes{}
		p.On("SystemResourceTypes").Return(r1, nil)
		err := c.Options("/resources", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceTypesError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 structs.ResourceTypes
		p.On("SystemResourceTypes").Return(nil, fmt.Errorf("err1"))
		err := c.Options("/resources", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceUnlink(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxResource
		r2 := structs.Resource{}
		p.On("SystemResourceUnlink", "resource1", "app1").Return(&r1, nil)
		err := c.Delete("/resources/resource1/links/app1", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceUnlinkError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Resource
		p.On("SystemResourceUnlink", "resource1", "app1").Return(nil, fmt.Errorf("err1"))
		err := c.Delete("/resources/resource1/links/app1", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemResourceUpdate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxResource
		r2 := structs.Resource{}
		opts := structs.ResourceUpdateOptions{
			Parameters: map[string]string{"k1": "v1", "k2": "v2"},
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"parameters": "k1=v1&k2=v2",
			},
		}
		p.On("SystemResourceUpdate", "resource1", opts).Return(&r1, nil)
		err := c.Put("/resources/resource1", ro, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemResourceUpdateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Resource
		opts := structs.ResourceUpdateOptions{
			Parameters: map[string]string{"k1": "v1", "k2": "v2"},
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"parameters": "k1=v1&k2=v2",
			},
		}
		p.On("SystemResourceUpdate", "resource1", opts).Return(nil, fmt.Errorf("err1"))
		err := c.Put("/resources/resource1", ro, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}
