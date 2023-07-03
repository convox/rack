package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/convox/rack/pkg/api"
	"github.com/convox/stdapi"
	"github.com/stretchr/testify/assert"
)

func TestAuthorize(t *testing.T) {
	s := &api.Server{}

	testData := []struct {
		c      *stdapi.Context
		access bool
	}{
		{
			c: func() *stdapi.Context {
				c := stdapi.NewContext(nil, httptest.NewRequest(http.MethodGet, "http://text.com", nil))
				api.SetReadRole(c)
				return c
			}(),
			access: true,
		},
		{
			c: func() *stdapi.Context {
				c := stdapi.NewContext(nil, httptest.NewRequest(http.MethodGet, "http://text.com", nil))
				return c
			}(),
			access: false,
		},
		{
			c: func() *stdapi.Context {
				c := stdapi.NewContext(nil, httptest.NewRequest(http.MethodPost, "http://text.com", nil))
				api.SetReadRole(c)
				return c
			}(),
			access: false,
		},
		{
			c: func() *stdapi.Context {
				c := stdapi.NewContext(nil, httptest.NewRequest(http.MethodPost, "http://text.com", nil))
				api.SetReadWriteRole(c)
				return c
			}(),
			access: true,
		},
	}

	for _, td := range testData {
		err := s.Authorize(func(c *stdapi.Context) error {
			return nil
		})(td.c)
		if td.access {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}
