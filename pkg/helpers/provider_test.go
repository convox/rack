package helpers_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAppEnvironment(t *testing.T) {
	release := structs.Release{
		Id:          "id",
		App:         "app1",
		Build:       "build",
		Env:         "env1=test1\nenv2=test2",
		Manifest:    "manifest",
		Description: "description",
	}

	provider := &mocksdk.Interface{}
	provider.On("ReleaseList", mock.Anything, mock.Anything).Return(func(app string, opts structs.ReleaseListOptions) structs.Releases {
		require.Equal(t, 1, *opts.Limit)
		require.Equal(t, release.App, app)
		return structs.Releases{release}
	}, nil)

	provider.On("ReleaseGet", mock.Anything, mock.Anything).Return(func(app, id string) *structs.Release {
		require.Equal(t, release.App, app)
		require.Equal(t, release.Id, id)
		return &release
	}, nil)

	envs, err := helpers.AppEnvironment(provider, release.App)
	require.NoError(t, err)
	require.Equal(t, release.Env, envs.String())
}

func TestAppManifest(t *testing.T) {
	release := structs.Release{
		Id:          "id",
		App:         "app1",
		Build:       "build",
		Env:         "env1=test1\nenv2=test2",
		Manifest:    "services:\r\n  web:\r\n    build: .\r\n    port: 80",
		Description: "description",
	}
	app := &structs.App{
		Name:    "app1",
		Release: release.Id,
	}

	provider := &mocksdk.Interface{}

	provider.On("AppGet", mock.Anything).Return(func(name string) *structs.App {
		require.Equal(t, app.Name, name)
		return app
	}, nil)

	provider.On("ReleaseGet", mock.Anything, mock.Anything).Return(func(app, id string) *structs.Release {
		require.Equal(t, release.App, app)
		require.Equal(t, release.Id, id)
		return &release
	}, nil)

	m, r, err := helpers.AppManifest(provider, release.App)
	require.NoError(t, err)
	require.Equal(t, &release, r)
	require.Equal(t, 1, len(m.Services))
	require.Equal(t, "web", m.Services[0].Name)
}

func TestReleaseLatest(t *testing.T) {
	release := structs.Release{
		Id:          "id",
		App:         "app1",
		Build:       "build",
		Env:         "env1=test1\nenv2=test2",
		Manifest:    "manifest",
		Description: "description",
	}

	provider := &mocksdk.Interface{}
	provider.On("ReleaseList", mock.Anything, mock.Anything).Return(func(app string, opts structs.ReleaseListOptions) structs.Releases {
		require.Equal(t, 1, *opts.Limit)
		require.Equal(t, release.App, app)
		return structs.Releases{release}
	}, nil)

	provider.On("ReleaseGet", mock.Anything, mock.Anything).Return(func(app, id string) *structs.Release {
		require.Equal(t, release.App, app)
		require.Equal(t, release.Id, id)
		return &release
	}, nil)

	got, err := helpers.ReleaseLatest(provider, release.App)
	require.NoError(t, err)
	require.Equal(t, &release, got)
}

func TestReleaseManifest(t *testing.T) {
	release := structs.Release{
		Id:          "id",
		App:         "app1",
		Build:       "build",
		Env:         "env1=test1\nenv2=test2",
		Manifest:    "services:\r\n  web:\r\n    build: .\r\n    port: 80",
		Description: "description",
	}

	provider := &mocksdk.Interface{}

	provider.On("ReleaseGet", mock.Anything, mock.Anything).Return(func(app, id string) *structs.Release {
		require.Equal(t, release.App, app)
		require.Equal(t, release.Id, id)
		return &release
	}, nil)

	m, r, err := helpers.ReleaseManifest(provider, release.App, release.Id)
	require.NoError(t, err)
	require.Equal(t, &release, r)
	require.Equal(t, 1, len(m.Services))
	require.Equal(t, "web", m.Services[0].Name)
}

func TestWaitForAppDeleted(t *testing.T) {
	provider := &mocksdk.Interface{}
	app := "app1"

	provider.On("AppGet", mock.Anything).Return(func(name string) *structs.App {
		require.Equal(t, app, name)
		return nil
	}, fmt.Errorf("no such app"))

	err := helpers.WaitForAppDeleted(provider, nil, app)
	require.NoError(t, err)
}

func TestWaitForAppRunning(t *testing.T) {
	provider := &mocksdk.Interface{}
	app := &structs.App{
		Name:   "app1",
		Status: "running",
	}

	provider.On("AppGet", mock.Anything).Return(func(name string) *structs.App {
		require.Equal(t, app.Name, name)
		return app
	}, nil)

	err := helpers.WaitForAppRunning(provider, app.Name)
	require.NoError(t, err)
}

func TestWaitForAppWithLogs(t *testing.T) {
	provider := &mocksdk.Interface{}
	app := &structs.App{
		Name:   "app1",
		Status: "running",
	}

	provider.On("AppGet", mock.Anything).Return(func(name string) *structs.App {
		require.Equal(t, app.Name, name)
		return app
	}, nil)

	provider.On("AppLogs", mock.Anything, mock.Anything).Return(func(name string, opts structs.LogsOptions) io.ReadCloser {
		require.Equal(t, app.Name, name)
		return &myReadCloser{
			bufio.NewReader(bytes.NewReader([]byte("test"))),
		}
	}, nil)

	err := helpers.WaitForAppWithLogs(provider, &bytes.Buffer{}, app.Name)
	require.NoError(t, err)
}

func TestWaitForProcessRunning(t *testing.T) {
	provider := &mocksdk.Interface{}
	p := &structs.Process{
		Id:     "id1",
		Name:   "p1",
		App:    "app1",
		Status: "running",
	}

	provider.On("ProcessGet", mock.Anything, mock.Anything).Return(func(app, pid string) *structs.Process {
		require.Equal(t, p.App, app)
		require.Equal(t, p.Id, pid)
		return p
	}, nil)

	err := helpers.WaitForProcessRunning(provider, nil, p.App, p.Id)
	require.NoError(t, err)
}

func TestWaitForRackRunning(t *testing.T) {
	provider := &mocksdk.Interface{}
	s := &structs.System{
		Name:   "s1",
		Status: "running",
	}

	provider.On("SystemGet").Return(func() *structs.System {
		return s
	}, nil)

	err := helpers.WaitForRackRunning(provider, nil)
	require.NoError(t, err)
}

func TestWaitForRackWithLogs(t *testing.T) {
	provider := &mocksdk.Interface{}
	s := &structs.System{
		Name:   "s1",
		Status: "running",
	}

	provider.On("SystemGet").Return(func() *structs.System {
		return s
	}, nil)

	provider.On("SystemLogs", mock.Anything).Return(func(opts structs.LogsOptions) io.ReadCloser {
		return &myReadCloser{
			bufio.NewReader(bytes.NewReader([]byte("test"))),
		}
	}, nil)

	err := helpers.WaitForRackWithLogs(provider, &bytes.Buffer{})
	require.NoError(t, err)
}

type myReadCloser struct {
	*bufio.Reader
}

func (mrc *myReadCloser) Close() error {
	// Noop
	return nil
}
