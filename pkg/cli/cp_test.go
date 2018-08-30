package cli_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCpUpload(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("FilesUpload", "app1", "0123456789", mock.Anything).Return(nil)

		res, err := testExecute(e, "cp -a app1 testdata/file 0123456789:/tmp/", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestCpUploadError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("FilesUpload", "app1", "0123456789", mock.Anything).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "cp -a app1 testdata/file 0123456789:/tmp/", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestCpDownload(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		tmpd, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		tmpf := filepath.Join(tmpd, "file")
		data, err := ioutil.ReadFile("testdata/file.tar")
		require.NoError(t, err)
		i.On("FilesDownload", "app1", "0123456789", "/tmp/file").Return(bytes.NewReader(data), nil)

		res, err := testExecute(e, fmt.Sprintf("cp -a app1 0123456789:/tmp/file %s", tmpf), nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
		odata, err := ioutil.ReadFile("testdata/file")
		require.NoError(t, err)
		ddata, err := ioutil.ReadFile(tmpf)
		require.NoError(t, err)
		require.Equal(t, odata, ddata)
	})
}

func TestCpDownloadError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		tmpd, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		tmpf := filepath.Join(tmpd, "file")
		i.On("FilesDownload", "app1", "0123456789", "/tmp/file").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, fmt.Sprintf("cp -a app1 0123456789:/tmp/file %s", tmpf), nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}
