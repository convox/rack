package start_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/convox/exec"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/start"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStart2(t *testing.T) {
	p := &structs.MockProvider{}

	mhc := mockHealthCheck(func(n int) int {
		return 200
	})
	defer mhc.Close()

	logs := "0000-00-00T00:00:00Z service/web/pid1 log1\n0000-00-00T00:00:00Z service/web/pid1 log2\n"

	p.On("AppGet", "app1").Return(&structs.App{Name: "app1", Generation: "2"}, nil)
	p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{{Id: "release1"}}, nil)
	p.On("ReleaseGet", "app1", "release1").Return(&structs.Release{}, nil)
	p.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)}).Return(ioutil.NopCloser(strings.NewReader(logs)), nil)

	e := &exec.MockInterface{}
	start.Exec = e

	e.On("Execute", "docker", "inspect", "httpd", "--format", "{{json .Config.Env}}").Return([]byte(`["FOO=bar","BAZ=qux"]`), nil)
	e.On("Execute", "docker", "inspect", "httpd", "--format", "{{.Config.WorkingDir}}").Return([]byte(`/app/foo`), nil)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	os.Chdir("testdata/httpd")
	defer os.Chdir(cwd)

	s := start.New()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	buf := bytes.Buffer{}

	opts := start.Options2{
		App:      "app1",
		Provider: p,
		Test:     true,
	}

	err = s.Start2(ctx, &buf, opts)
	require.NoError(t, err)

	require.Equal(t,
		[]string{
			"<color3>web   </color3> | log1",
			"<color3>web   </color3> | log2",
			"<system>convox</system> | stopping",
		},
		strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"),
	)

	p.AssertExpectations(t)
	e.AssertExpectations(t)
}

func TestStart2Options(t *testing.T) {
	helpers.ProviderWaitDuration = 1

	p := &structs.MockProvider{}

	appLogs := "0000-00-00T00:00:00Z service/web/pid1 log1\n0000-00-00T00:00:00Z service/web/pid1 log2\n"
	buildLogs := "build1\nbuild2\n"

	p.On("AppGet", "app1").Return(&structs.App{Name: "app1", Generation: "2", Release: "old", Status: "running"}, nil)
	p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{{Id: "release1"}}, nil)
	p.On("ReleaseGet", "app1", "release1").Return(&structs.Release{}, nil)
	p.On("ObjectStore", "app1", "", mock.Anything, structs.ObjectStoreOptions{}).Return(&structs.Object{Url: "object://app1/object1.tgz"}, nil)
	p.On("BuildCreate", "app1", "object://app1/object1.tgz", structs.BuildCreateOptions{Development: options.Bool(true), Manifest: options.String("convox2.yml")}).Return(&structs.Build{Id: "build1"}, nil)
	p.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(ioutil.NopCloser(strings.NewReader(buildLogs)), nil)
	p.On("BuildGet", "app1", "build1").Return(&structs.Build{Id: "build1", Release: "release1", Status: "complete"}, nil)
	p.On("ReleasePromote", "app1", "release1", structs.ReleasePromoteOptions{Development: options.Bool(true), Force: options.Bool(true), Idle: options.Bool(false), Min: options.Int(0), Timeout: options.Int(300)}).Return(nil)
	p.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)}).Return(ioutil.NopCloser(strings.NewReader(appLogs)), nil).Once()
	p.On("ReleasePromote", "app1", "old", structs.ReleasePromoteOptions{Development: options.Bool(false), Force: options.Bool(true)}).Return(nil)

	e := &exec.MockInterface{}
	start.Exec = e

	e.On("Execute", "docker", "inspect", "httpd", "--format", "{{json .Config.Env}}").Return([]byte(`["FOO=bar","BAZ=qux"]`), nil)
	e.On("Execute", "docker", "inspect", "httpd", "--format", "{{.Config.WorkingDir}}").Return([]byte(`/app/foo`), nil)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	os.Chdir("testdata/httpd")
	defer os.Chdir(cwd)

	s := start.New()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	buf := bytes.Buffer{}

	opts := start.Options2{
		App:      "app1",
		Build:    true,
		Cache:    true,
		Manifest: "convox2.yml",
		Provider: p,
		Sync:     true,
		Test:     true,
	}

	err = s.Start2(ctx, &buf, opts)
	require.NoError(t, err)

	require.Equal(t,
		[]string{
			"<system>build </system> | uploading source",
			"<system>build </system> | starting build",
			"<system>build </system> | build1",
			"<system>build </system> | build2",
			"<color3>web   </color3> | log1",
			"<color3>web   </color3> | log2",
			"<system>convox</system> | stopping",
		},
		strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"),
	)

	p.AssertExpectations(t)
	e.AssertExpectations(t)
}
