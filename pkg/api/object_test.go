package api_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fxObject = structs.Object{
	Url: "https://example.org/path",
}

func TestObjectDelete(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("ObjectDelete", "app1", "path/object1.ext").Return(nil)
		err := c.Delete("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestObjectDeleteError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("ObjectDelete", "app1", "path/object1.ext").Return(fmt.Errorf("err1"))
		err := c.Delete("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestObjectExists(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var exists bool
		p.On("ObjectExists", "app1", "path/object1.ext").Return(true, nil)
		err := c.Head("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{}, &exists)
		require.NoError(t, err)
		require.Equal(t, true, exists)
	})
}

func TestObjectExistsError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var exists bool
		p.On("ObjectExists", "app1", "path/object1.ext").Return(false, fmt.Errorf("err1"))
		err := c.Head("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{}, &exists)
		require.EqualError(t, err, "response status 500")
		require.Equal(t, false, exists)
	})
}

func TestObjectFetch(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		d1 := []byte("test")
		r1 := ioutil.NopCloser(bytes.NewReader(d1))
		p.On("ObjectFetch", "app1", "path/object1.ext").Return(r1, nil)
		res, err := c.GetStream("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{})
		require.NoError(t, err)
		require.NotNil(t, res)
		defer res.Body.Close()
		d2, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, d1, d2)
	})
}

func TestObjectFetchError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("ObjectFetch", "app1", "path/object1.ext").Return(nil, fmt.Errorf("err1"))
		res, err := c.GetStream("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{})
		require.EqualError(t, err, "err1")
		require.Nil(t, res)
	})
}

func TestObjectList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		o1 := []string{"object1", "object2"}
		o2 := []string{}
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"prefix": "path",
			},
		}
		p.On("ObjectList", "app1", "path").Return(o1, nil)
		err := c.Get("/apps/app1/objects", ro, &o2)
		require.NoError(t, err)
		require.Equal(t, o1, o2)
	})
}

func TestObjectListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var o1 []string
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"prefix": "path",
			},
		}
		p.On("ObjectList", "app1", "path").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/objects", ro, &o1)
		require.EqualError(t, err, "err1")
		require.Nil(t, o1)
	})
}

func TestObjectStore(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		o1 := fxObject
		o2 := structs.Object{}
		opts := structs.ObjectStoreOptions{
			Public: options.Bool(true),
		}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("data"),
			Headers: stdsdk.Headers{
				"Public": "true",
			},
		}
		p.On("ObjectStore", "app1", "path/object1.ext", mock.Anything, opts).Return(&o1, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "data", string(data))
		})
		err := c.Post("/apps/app1/objects/path/object1.ext", ro, &o2)
		require.NoError(t, err)
		require.Equal(t, o1, o2)
	})
}

func TestObjectStoreError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var o1 *structs.Object
		p.On("ObjectStore", "app1", "path/object1.ext", mock.Anything, structs.ObjectStoreOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps/app1/objects/path/object1.ext", stdsdk.RequestOptions{}, &o1)
		require.EqualError(t, err, "err1")
		require.Nil(t, o1)
	})
}
