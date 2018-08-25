package api_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFilesDelete(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"files": "file1,file2",
			},
		}
		p.On("FilesDelete", "app1", "pid1", []string{"file1", "file2"}).Return(nil)
		err := c.Delete("/apps/app1/processes/pid1/files", opts, nil)
		require.NoError(t, err)
	})
}

func TestFilesDeleteError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"files": "file1,file2",
			},
		}
		p.On("FilesDelete", "app1", "pid1", []string{"file1", "file2"}).Return(fmt.Errorf("err1"))
		err := c.Delete("/apps/app1/processes/pid1/files", opts, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestFilesUpload(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := stdsdk.RequestOptions{
			Body: strings.NewReader("data"),
		}
		p.On("FilesUpload", "app1", "pid1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "data", string(data))
		})
		err := c.Post("/apps/app1/processes/pid1/files", opts, nil)
		require.NoError(t, err)
	})
}

func TestFilesUploadError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("FilesUpload", "app1", "pid1", mock.Anything).Return(fmt.Errorf("err1"))
		err := c.Post("/apps/app1/processes/pid1/files", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
