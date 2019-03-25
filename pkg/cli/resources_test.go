package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResources(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceList", "app1").Return(structs.Resources{*fxResource(), *fxResource()}, nil)

		res, err := testExecute(e, "resources -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"NAME       TYPE  URL                     ",
			"resource1  type  https://example.org/path",
			"resource1  type  https://example.org/path",
		})
	})
}

func TestResourcesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceList", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "resources -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestResourcesInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceGet", "app1", "resource1").Return(fxResource(), nil)

		res, err := testExecute(e, "resources info resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name  resource1",
			"Type  type",
			"URL   https://example.org/path",
		})
	})
}

func TestResourcesInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceGet", "app1", "resource1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "resources info resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestResourcesProxy(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceGet", "app1", "resource1").Return(fxResource(), nil)
		i.On("Proxy", "example.org", 443, mock.Anything, structs.ProxyOptions{TLS: options.Bool(false)}).Return(nil).Run(func(args mock.Arguments) {
			buf := make([]byte, 2)
			rwc := args.Get(2).(io.ReadWriteCloser)
			n, err := rwc.Read(buf)
			require.NoError(t, err)
			require.Equal(t, 2, n)
			require.Equal(t, "in", string(buf))
			n, err = rwc.Write([]byte("out"))
			require.NoError(t, err)
			require.Equal(t, 3, n)
			rwc.Close()
		})

		port := rand.Intn(30000) + 10000

		ch := make(chan *result)

		go func() {
			res, _ := testExecute(e, fmt.Sprintf("resources proxy resource1 -a app1 -p %d", port), nil)
			ch <- res
		}()

		time.Sleep(2 * time.Second)

		cn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		require.NoError(t, err)

		cn.Write([]byte("in"))

		data, err := ioutil.ReadAll(cn)
		require.NoError(t, err)
		require.Equal(t, "out", string(data))

		cli.ProxyCloser <- nil

		res := <-ch

		require.NotNil(t, res)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			fmt.Sprintf("proxying localhost:%d to example.org:443", port),
			fmt.Sprintf("connect: %d", port),
		})
	})
}

func TestResourcesUrl(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceGet", "app1", "resource1").Return(fxResource(), nil)

		res, err := testExecute(e, "resources url resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"https://example.org/path"})
	})
}

func TestResourcesUrlError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ResourceGet", "app1", "resource1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "resources url resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackResources(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceList").Return(structs.Resources{*fxResource(), *fxResource()}, nil)

		res, err := testExecute(e, "rack resources", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"NAME       TYPE  STATUS",
			"resource1  type  status",
			"resource1  type  status",
		})
	})
}

func TestRackResourcesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackResourcesCreate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ResourceCreateOptions{Name: options.String("name1"), Parameters: map[string]string{"Foo": "bar", "Baz": "quux"}}
		i.On("SystemResourceCreate", "type1", opts).Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources create type1 -n name1 Foo=bar Baz=quux", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Creating resource... OK, resource1"})
	})
}

func TestRackResourcesCreateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ResourceCreateOptions{Name: options.String("name1"), Parameters: map[string]string{"Foo": "bar", "Baz": "quux"}}
		i.On("SystemResourceCreate", "type1", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources create type1 -n name1 Foo=bar Baz=quux", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Creating resource... "})
	})
}

func TestRackResourcesCreateClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.ResourceCreateOptions{Name: options.String("name1"), Parameters: map[string]string{"Foo": "bar", "Baz": "quux"}}
		i.On("ResourceCreateClassic", "type1", opts).Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources create type1 -n name1 Foo=bar Baz=quux", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Creating resource... OK, resource1"})
	})
}

func TestRackResourcesDelete(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceDelete", "resource1").Return(nil)

		res, err := testExecute(e, "rack resources delete resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Deleting resource... OK"})
	})
}

func TestRackResourcesDeleteError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceDelete", "resource1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources delete resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Deleting resource... "})
	})
}

func TestRackResourcesInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceGet", "resource1").Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources info resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name     resource1",
			"Type     type",
			"Status   status",
			"Options  Url=https://other.example.org/path",
			"         k1=v1",
			"         k2=v2",
			"URL      https://example.org/path",
			"Apps     app1, app1",
		})
	})
}

func TestRackResourcesInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceGet", "resource1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources info resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackResourcesLink(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceLink", "resource1", "app1").Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources link resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Linking to app1... OK"})
	})
}

func TestRackResourcesLinkError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceLink", "resource1", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources link resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Linking to app1... "})
	})
}

func TestRackResourcesOptions(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceTypes").Return(structs.ResourceTypes{fxResourceType()}, nil)

		res, err := testExecute(e, "rack resources options type1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"NAME    DEFAULT  DESCRIPTION",
			"Param1  def1     desc1      ",
			"Param2  def2     desc2      ",
		})
	})
}

func TestRackResourcesOptionsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceTypes").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources options type1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackResourcesProxy(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceGet", "resource1").Return(fxResource(), nil)
		i.On("Proxy", "example.org", 443, mock.Anything, structs.ProxyOptions{TLS: options.Bool(false)}).Return(nil).Run(func(args mock.Arguments) {
			buf := make([]byte, 2)
			rwc := args.Get(2).(io.ReadWriteCloser)
			n, err := rwc.Read(buf)
			require.NoError(t, err)
			require.Equal(t, 2, n)
			require.Equal(t, "in", string(buf))
			n, err = rwc.Write([]byte("out"))
			require.NoError(t, err)
			require.Equal(t, 3, n)
			rwc.Close()
		})

		port := rand.Intn(30000) + 10000

		ch := make(chan *result)

		go func() {
			res, _ := testExecute(e, fmt.Sprintf("rack resources proxy resource1 -p %d", port), nil)
			ch <- res
		}()

		time.Sleep(50 * time.Millisecond)

		cn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		require.NoError(t, err)

		cn.Write([]byte("in"))

		data, err := ioutil.ReadAll(cn)
		require.NoError(t, err)
		require.Equal(t, "out", string(data))

		cli.ProxyCloser <- nil

		res := <-ch

		require.NotNil(t, res)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			fmt.Sprintf("proxying localhost:%d to example.org:443", port),
			fmt.Sprintf("connect: %d", port),
		})
	})
}

func TestRackResourcesTypes(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceTypes").Return(structs.ResourceTypes{fxResourceType(), fxResourceType()}, nil)

		res, err := testExecute(e, "rack resources types", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"TYPE ",
			"type1",
			"type1",
		})
	})
}

func TestRackResourcesTypesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceTypes").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources types", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackResourcesUnlink(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceUnlink", "resource1", "app1").Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources unlink resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Unlinking from app1... OK"})
	})
}

func TestRackResourcesUnlinkError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceUnlink", "resource1", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources unlink resource1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Unlinking from app1... "})
	})
}

func TestRackResourcesUpdate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ResourceUpdateOptions{Parameters: map[string]string{"Foo": "bar", "Baz": "quux"}}
		i.On("SystemResourceUpdate", "resource1", opts).Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources update resource1 Foo=bar Baz=quux", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating resource... OK"})
	})
}

func TestRackResourcesUpdateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ResourceUpdateOptions{Parameters: map[string]string{"Foo": "bar", "Baz": "quux"}}
		i.On("SystemResourceUpdate", "resource1", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources update resource1 Foo=bar Baz=quux", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Updating resource... "})
	})
}

func TestRackResourcesUpdateClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.ResourceUpdateOptions{Parameters: map[string]string{"Foo": "bar", "Baz": "quux"}}
		i.On("ResourceUpdateClassic", "resource1", opts).Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources update resource1 Foo=bar Baz=quux", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating resource... OK"})
	})
}

func TestRackResourcesUrl(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceGet", "resource1").Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources url resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"https://example.org/path"})
	})
}

func TestRackResourcesUrlError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemResourceGet", "resource1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack resources url resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackResourcesUrlClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("SystemResourceGetClassic", "resource1").Return(fxResource(), nil)

		res, err := testExecute(e, "rack resources url resource1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"https://other.example.org/path"})
	})
}
