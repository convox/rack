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

	logs := "0000-00-00 00:00:00 service/web/pid1 log1\n0000-00-00 00:00:00 service/web/pid1 log2\n"

	p.On("AppGet", "app1").Return(&structs.App{Name: "app1", Generation: "2"}, nil)
	p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{{Id: "release1"}}, nil)
	p.On("ReleaseGet", "app1", "release1").Return(&structs.Release{}, nil)
	p.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)}).Return(ioutil.NopCloser(strings.NewReader(logs)), nil)
	p.On("ServiceList", "app1").Return(structs.Services{{Name: "web", Domain: mhc.Listener.Addr().String()}}, nil)
	p.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(structs.Processes{{Id: "process1"}, {Id: "process2"}}, nil)
	p.On("ProcessStop", "app1", "process1").Return(nil)
	p.On("ProcessStop", "app1", "process2").Return(nil)

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
			"<system>convox</system> | starting health check for <service>web</service> on path <setting>/</setting> with <setting>5</setting>s interval, <setting>5</setting>s grace",
			"<color3>web   </color3> | log1",
			"<color3>web   </color3> | log2",
			"<system>convox</system> | health check <service>web</service>: <ok>200</ok>",
			"<system>convox</system> | stopping process1",
			"<system>convox</system> | stopping process2",
		},
		strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"),
	)
}

func TestStart2Options(t *testing.T) {
	p := &structs.MockProvider{}

	mhc := mockHealthCheck(func(n int) int {
		return 200
	})
	defer mhc.Close()

	appLogs := "0000-00-00 00:00:00 service/web/pid1 log1\n0000-00-00 00:00:00 service/web/pid1 log2\n"
	buildLogs := "build1\nbuild2\n"

	p.On("AppGet", "app1").Return(&structs.App{Name: "app1", Generation: "2"}, nil)
	p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{{Id: "release1"}}, nil)
	p.On("ReleaseGet", "app1", "release1").Return(&structs.Release{}, nil)
	p.On("ObjectStore", "app1", "", mock.Anything, structs.ObjectStoreOptions{}).Return(&structs.Object{Url: "object://app1/object1.tgz"}, nil)
	p.On("BuildCreate", "app1", "object://app1/object1.tgz", structs.BuildCreateOptions{Development: options.Bool(true), Manifest: options.String("convox2.yml")}).Return(&structs.Build{Id: "build1"}, nil)
	p.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(ioutil.NopCloser(strings.NewReader(buildLogs)), nil)
	p.On("BuildGet", "app1", "build1").Return(&structs.Build{Id: "build1", Release: "release1", Status: "complete"}, nil)
	p.On("ReleasePromote", "app1", "release1").Return(nil)
	p.On("ServiceList", "app1").Return(structs.Services{{Name: "web", Domain: mhc.Listener.Addr().String()}}, nil)
	p.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)}).Return(ioutil.NopCloser(strings.NewReader(appLogs)), nil)
	p.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(structs.Processes{{Id: "process1"}, {Id: "process2"}}, nil)
	p.On("ProcessStop", "app1", "process1").Return(nil)
	p.On("ProcessStop", "app1", "process2").Return(nil)

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
			"<system>convox</system> | starting health check for <service>web</service> on path <setting>/</setting> with <setting>5</setting>s interval, <setting>5</setting>s grace",
			"<color3>web   </color3> | log1",
			"<color3>web   </color3> | log2",
			"<system>convox</system> | health check <service>web</service>: <ok>200</ok>",
			"<system>convox</system> | stopping process1",
			"<system>convox</system> | stopping process2",
		},
		strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"),
	)
}

func TestStart2Health(t *testing.T) {
	p := &structs.MockProvider{}

	// 503, 200, 200, 500, 500, 200, 200...
	mhc := mockHealthCheck(func(n int) int {
		if n < 1 {
			return 503
		} else if n >= 3 && n < 5 {
			return 500
		} else {
			return 200
		}
	})
	defer mhc.Close()

	logs := "0000-00-00 00:00:00 service/web/pid1 log1\n0000-00-00 00:00:00 service/web/pid1 log2\n"

	p.On("AppGet", "app1").Return(&structs.App{Name: "app1", Generation: "2"}, nil)
	p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{{Id: "release1"}}, nil)
	p.On("ReleaseGet", "app1", "release1").Return(&structs.Release{}, nil)
	p.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1 * time.Second)}).Return(ioutil.NopCloser(strings.NewReader(logs)), nil)
	p.On("ServiceList", "app1").Return(structs.Services{{Name: "web", Domain: mhc.Listener.Addr().String()}}, nil)
	p.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(structs.Processes{{Id: "process1"}, {Id: "process2"}}, nil)
	p.On("ProcessStop", "app1", "process1").Return(nil)
	p.On("ProcessStop", "app1", "process2").Return(nil)

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

	// less than 6 checks would mean we don't test properly
	if mhc.Count() < 6 {
		require.Fail(t, "expected at least 6 healthchecks")
	}

	require.Equal(t,
		[]string{
			"<system>convox</system> | starting health check for <service>web</service> on path <setting>/</setting> with <setting>5</setting>s interval, <setting>5</setting>s grace",
			"<color3>web   </color3> | log1",
			"<color3>web   </color3> | log2",
			"<system>convox</system> | health check <service>web</service>: <fail>503</fail>",
			"<system>convox</system> | health check <service>web</service>: <ok>200</ok>",
			"<system>convox</system> | health check <service>web</service>: <fail>500</fail>",
			"<system>convox</system> | health check <service>web</service>: <fail>500</fail>",
			"<system>convox</system> | health check <service>web</service>: <ok>200</ok>",
			"<system>convox</system> | stopping process1",
			"<system>convox</system> | stopping process2",
		},
		strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"),
	)
}
