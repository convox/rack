package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxRegistry = structs.Registry{
	Server:   "registry1",
	Username: "username",
	Password: "password",
}

func TestRegistryAdd(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := fxRegistry
		r2 := structs.Registry{}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"password": "password",
				"server":   "registry1",
				"username": "username",
			},
		}
		p.On("RegistryAdd", "registry1", "username", "password").Return(&r1, nil)
		err := c.Post("/registries", ro, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestRegistryAddError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 *structs.Registry
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"password": "password",
				"server":   "registry1",
				"username": "username",
			},
		}
		p.On("RegistryAdd", "registry1", "username", "password").Return(nil, fmt.Errorf("err1"))
		err := c.Post("/registries", ro, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestRegistryList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := structs.Registries{fxRegistry, fxRegistry}
		r2 := structs.Registries{}
		p.On("RegistryList").Return(r1, nil)
		err := c.Get("/registries", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestRegistryListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 structs.Registries
		p.On("RegistryList").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/registries", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestRegistryRemove(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("RegistryRemove", "registry1").Return(nil)
		err := c.Delete("/registries/registry1", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestRegistryRemoveError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("RegistryRemove", "registry1").Return(fmt.Errorf("err1"))
		err := c.Delete("/registries/registry1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
