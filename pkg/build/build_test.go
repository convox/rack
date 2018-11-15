package build_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/convox/exec"
	"github.com/convox/rack/pkg/build"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildGeneration2(t *testing.T) {
	opts := build.Options{
		App:        "app1",
		Auth:       "{}",
		Cache:      true,
		Generation: "2",
		Id:         "build1",
		Rack:       "rack1",
		Source:     "object://app1/object.tgz",
	}

	testBuild(t, opts, func(b *build.Build, p *structs.MockProvider, e *exec.MockInterface, out *bytes.Buffer) {
		p.On("BuildGet", "app1", "build1").Return(fxBuildStarted(), nil).Once()
		bdata, err := ioutil.ReadFile("testdata/httpd.tgz")
		require.NoError(t, err)
		p.On("ObjectFetch", "app1", "/object.tgz").Return(ioutil.NopCloser(bytes.NewReader(bdata)), nil)
		mdata, err := ioutil.ReadFile("testdata/httpd/convox.yml")
		require.NoError(t, err)
		p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{*fxRelease()}, nil)
		p.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		e.On("Run", mock.Anything, "docker", "build", "-t", "049f26f1b03bfca2e3af367d481a7bf1a94564ba", "-f", "Dockerfile", ".").Return(nil).Run(func(args mock.Arguments) {
			fmt.Fprintf(args.Get(0).(io.Writer), "build1\nbuild2\n")
		})
		e.On("Execute", "docker", "pull", "httpd").Return([]byte("pulling\n"), nil)
		e.On("Execute", "docker", "tag", "httpd", "rack1/app1:web.build1").Return([]byte("tagging\n"), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web.build1", "--format", "{{json .Config.Cmd}}").Return([]byte(`["command1"]`), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web.build1", "--format", "{{json .Config.Entrypoint}}").Return([]byte(`["entrypoint1"]`), nil)
		e.On("Execute", "cp", "/go/bin/convox-env", mock.AnythingOfType("string")).Return([]byte("copying\n"), nil)
		e.On("Execute", "docker", "build", "-t", "rack1/app1:web.build1", mock.AnythingOfType("string")).Return([]byte("building convox-env\n"), nil)
		e.On("Execute", "docker", "tag", "049f26f1b03bfca2e3af367d481a7bf1a94564ba", "rack1/app1:web2.build1").Return([]byte("tagging\n"), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web2.build1", "--format", "{{json .Config.Cmd}}").Return([]byte(`["command2"]`), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web2.build1", "--format", "{{json .Config.Entrypoint}}").Return([]byte(`["entrypoint2"]`), nil)
		e.On("Execute", "docker", "build", "-t", "rack1/app1:web2.build1", mock.AnythingOfType("string")).Return([]byte("building convox-env\n"), nil)
		p.On("ObjectStore", "app1", "build/build1/logs", mock.Anything, structs.ObjectStoreOptions{}).Return(fxObject(), nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "Building: .\nbuild1\nbuild2\nRunning: docker pull httpd\nRunning: docker tag 049f26f1b03bfca2e3af367d481a7bf1a94564ba rack1/app1:web2.build1\nInjecting: convox-env\nRunning: docker tag httpd rack1/app1:web.build1\nInjecting: convox-env\n", string(data))
		})
		p.On("BuildUpdate", "app1", "build1", mock.Anything).Return(fxBuildStarted(), nil).Run(func(args mock.Arguments) {
			opts := args.Get(2).(structs.BuildUpdateOptions)
			if opts.Ended != nil {
				require.False(t, opts.Ended.IsZero())
			}
			if opts.Logs != nil {
				require.NotNil(t, opts.Logs)
			}
			if opts.Manifest != nil {
				require.Equal(t, string(mdata), *opts.Manifest)
			}
		})
		p.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Build: options.String("build1")}).Return(fxRelease2(), nil)
		p.On("EventSend", "build:create", structs.EventSendOptions{Data: map[string]string{"app": "app1", "id": "build1", "release_id": "release2"}}).Return(nil)

		err = b.Execute()
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"Building: .",
				"build1",
				"build2",
				"Running: docker pull httpd",
				"Running: docker tag 049f26f1b03bfca2e3af367d481a7bf1a94564ba rack1/app1:web2.build1",
				"Injecting: convox-env",
				"Running: docker tag httpd rack1/app1:web.build1",
				"Injecting: convox-env",
			},
			strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"),
		)
	})
}

func TestBuildGeneration2Development(t *testing.T) {
	opts := build.Options{
		App:         "app1",
		Auth:        "{}",
		Cache:       true,
		Development: true,
		Generation:  "2",
		Id:          "build1",
		Rack:        "rack1",
		Source:      "object://app1/object.tgz",
	}

	testBuild(t, opts, func(b *build.Build, p *structs.MockProvider, e *exec.MockInterface, out *bytes.Buffer) {
		p.On("BuildGet", "app1", "build1").Return(fxBuildStarted(), nil).Once()
		bdata, err := ioutil.ReadFile("testdata/httpd-dev.tgz")
		require.NoError(t, err)
		p.On("ObjectFetch", "app1", "/object.tgz").Return(ioutil.NopCloser(bytes.NewReader(bdata)), nil)
		mdata, err := ioutil.ReadFile("testdata/httpd-dev/convox.yml")
		require.NoError(t, err)
		p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{*fxRelease()}, nil)
		p.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		e.On("Run", mock.Anything, "docker", "build", "-t", "049f26f1b03bfca2e3af367d481a7bf1a94564ba", "-f", "Dockerfile", "--target", "development", ".").Return(nil).Run(func(args mock.Arguments) {
			fmt.Fprintf(args.Get(0).(io.Writer), "build1\nbuild2\n")
		})
		e.On("Execute", "docker", "tag", "049f26f1b03bfca2e3af367d481a7bf1a94564ba", "rack1/app1:web.build1").Return([]byte("tagging\n"), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web.build1", "--format", "{{json .Config.Cmd}}").Return([]byte(`["command2"]`), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web.build1", "--format", "{{json .Config.Entrypoint}}").Return([]byte(`["entrypoint2"]`), nil)
		e.On("Execute", "cp", "/go/bin/convox-env", mock.AnythingOfType("string")).Return([]byte("copying\n"), nil)
		e.On("Execute", "docker", "build", "-t", "rack1/app1:web.build1", mock.AnythingOfType("string")).Return([]byte("building convox-env\n"), nil)
		p.On("ObjectStore", "app1", "build/build1/logs", mock.Anything, structs.ObjectStoreOptions{}).Return(fxObject(), nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "Building: .\nbuild1\nbuild2\nRunning: docker tag 049f26f1b03bfca2e3af367d481a7bf1a94564ba rack1/app1:web.build1\nInjecting: convox-env\n", string(data))
		})
		p.On("BuildUpdate", "app1", "build1", mock.Anything).Return(fxBuildStarted(), nil).Run(func(args mock.Arguments) {
			opts := args.Get(2).(structs.BuildUpdateOptions)
			if opts.Ended != nil {
				require.False(t, opts.Ended.IsZero())
			}
			if opts.Logs != nil {
				require.NotNil(t, opts.Logs)
			}
			if opts.Manifest != nil {
				require.Equal(t, string(mdata), *opts.Manifest)
			}
		})
		p.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Build: options.String("build1")}).Return(fxRelease2(), nil)
		p.On("EventSend", "build:create", structs.EventSendOptions{Data: map[string]string{"app": "app1", "id": "build1", "release_id": "release2"}}).Return(nil)

		err = b.Execute()
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"Building: .",
				"build1",
				"build2",
				"Running: docker tag 049f26f1b03bfca2e3af367d481a7bf1a94564ba rack1/app1:web.build1",
				"Injecting: convox-env",
			},
			strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"),
		)
	})
}

func TestBuildGeneration2Failure(t *testing.T) {
	opts := build.Options{
		App:        "app1",
		Auth:       "{}",
		Cache:      true,
		Generation: "2",
		Id:         "build1",
		Rack:       "rack1",
		Source:     "object://app1/object.tgz",
	}

	testBuild(t, opts, func(b *build.Build, p *structs.MockProvider, e *exec.MockInterface, out *bytes.Buffer) {
		p.On("BuildGet", "app1", "build1").Return(fxBuildStarted(), nil)
		p.On("ObjectFetch", "app1", "/object.tgz").Return(nil, fmt.Errorf("err1"))
		p.On("ObjectStore", "app1", "build/build1/logs", mock.Anything, structs.ObjectStoreOptions{}).Return(fxObject(), nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "ERROR: err1\n", string(data))
		})
		p.On("BuildUpdate", "app1", "build1", mock.Anything).Return(fxBuildStarted(), nil).Run(func(args mock.Arguments) {
			opts := args.Get(2).(structs.BuildUpdateOptions)
			require.NotNil(t, opts.Ended)
			require.False(t, opts.Ended.IsZero())
			require.NotNil(t, opts.Logs)
			require.Equal(t, "object://app1/build/build1/logs", *opts.Logs)
			require.NotNil(t, opts.Status)
			require.Equal(t, "failed", *opts.Status)
		})
		p.On("EventSend", "build:create", structs.EventSendOptions{Data: map[string]string{"app": "app1", "id": "build1"}, Error: options.String("err1")}).Return(nil)

		err := b.Execute()
		require.EqualError(t, err, "err1")

		require.Equal(t,
			[]string{
				"ERROR: err1",
			},
			strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"),
		)
	})
}

func TestBuildGeneration2Options(t *testing.T) {
	opts := build.Options{
		App:         "app1",
		Auth:        `{"host1":{"username":"user1","password":"pass1"}}`,
		Cache:       false,
		Development: true,
		Generation:  "2",
		Id:          "build1",
		Manifest:    "convox2.yml",
		Push:        "push1",
		Rack:        "rack1",
		Source:      "object://app1/object.tgz",
	}

	testBuild(t, opts, func(b *build.Build, p *structs.MockProvider, e *exec.MockInterface, out *bytes.Buffer) {
		p.On("BuildGet", "app1", "build1").Return(fxBuildStarted(), nil).Once()
		e.On("Stream", mock.Anything, mock.Anything, "docker", "login", "-u", "user1", "--password-stdin", "host1").Return(nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "pass1", string(data))
			fmt.Fprintf(args.Get(0).(io.Writer), "login-success\n")
		})
		bdata, err := ioutil.ReadFile("testdata/httpd.tgz")
		require.NoError(t, err)
		p.On("ObjectFetch", "app1", "/object.tgz").Return(ioutil.NopCloser(bytes.NewReader(bdata)), nil)
		mdata, err := ioutil.ReadFile("testdata/httpd/convox2.yml")
		require.NoError(t, err)
		// p.On("BuildUpdate", "app1", "build1", structs.BuildUpdateOptions{Manifest: options.String(string(mdata))}).Return(fxBuildStarted(), nil).Once()
		p.On("ReleaseList", "app1", structs.ReleaseListOptions{Limit: options.Int(1)}).Return(structs.Releases{*fxRelease()}, nil)
		p.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		e.On("Run", mock.Anything, "docker", "build", "--no-cache", "-t", "63b602b07e75429dbf1ab14132f20c9e5a649f2f", "-f", "Dockerfile2", "--build-arg", "FOO=bar", ".").Return(nil).Run(func(args mock.Arguments) {
			fmt.Fprintf(args.Get(0).(io.Writer), "build1\nbuild2\n")
		})
		e.On("Execute", "docker", "tag", "63b602b07e75429dbf1ab14132f20c9e5a649f2f", "rack1/app1:web.build1").Return([]byte("tagging\n"), nil)
		e.On("Execute", "docker", "tag", "rack1/app1:web.build1", "push1:web.build1").Return([]byte("tagging\n"), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web.build1", "--format", "{{json .Config.Cmd}}").Return([]byte(`["command2"]`), nil)
		e.On("Execute", "docker", "inspect", "rack1/app1:web.build1", "--format", "{{json .Config.Entrypoint}}").Return([]byte(`["entrypoint2"]`), nil)
		e.On("Execute", "cp", "/go/bin/convox-env", mock.AnythingOfType("string")).Return([]byte("copying\n"), nil)
		e.On("Execute", "docker", "build", "-t", "rack1/app1:web.build1", mock.AnythingOfType("string")).Return([]byte("building convox-env\n"), nil)
		e.On("Execute", "docker", "push", "push1:web.build1").Return([]byte("pushing\n"), nil)
		p.On("ObjectStore", "app1", "build/build1/logs", mock.Anything, structs.ObjectStoreOptions{}).Return(fxObject(), nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "Authenticating host1: login-success\nBuilding: .\nbuild1\nbuild2\nRunning: docker tag 63b602b07e75429dbf1ab14132f20c9e5a649f2f rack1/app1:web.build1\nInjecting: convox-env\nRunning: docker tag rack1/app1:web.build1 push1:web.build1\nRunning: docker push push1:web.build1\n", string(data))
		})
		p.On("BuildUpdate", "app1", "build1", mock.Anything).Return(fxBuildStarted(), nil).Run(func(args mock.Arguments) {
			opts := args.Get(2).(structs.BuildUpdateOptions)
			if opts.Ended != nil {
				require.False(t, opts.Ended.IsZero())
			}
			if opts.Logs != nil {
				require.NotNil(t, opts.Logs)
			}
			if opts.Manifest != nil {
				require.Equal(t, string(mdata), *opts.Manifest)
			}
		})
		p.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Build: options.String("build1")}).Return(fxRelease2(), nil)
		p.On("EventSend", "build:create", structs.EventSendOptions{Data: map[string]string{"app": "app1", "id": "build1", "release_id": "release2"}}).Return(nil)

		err = b.Execute()
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"Authenticating host1: login-success",
				"Building: .",
				"build1",
				"build2",
				"Running: docker tag 63b602b07e75429dbf1ab14132f20c9e5a649f2f rack1/app1:web.build1",
				"Injecting: convox-env",
				"Running: docker tag rack1/app1:web.build1 push1:web.build1",
				"Running: docker push push1:web.build1",
			},
			strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"),
		)
	})
}

func fxBuildStarted() *structs.Build {
	return &structs.Build{
		Id:          "build1",
		Description: "desc",
		Started:     time.Now().UTC(),
		Status:      "started",
	}
}

func fxObject() *structs.Object {
	return &structs.Object{
		Url: "object://app1/build/build1/logs",
	}
}

func fxRelease() *structs.Release {
	return &structs.Release{
		Id:       "release1",
		App:      "app1",
		Build:    "build1",
		Env:      "FOO=bar\nBAZ=quux",
		Manifest: "services:\n  web:\n    build: .",
		Created:  time.Now().UTC().Add(-49 * time.Hour),
	}
}

func fxRelease2() *structs.Release {
	return &structs.Release{
		Id:       "release2",
		App:      "app1",
		Build:    "build1",
		Env:      "FOO=bar\nBAZ=quux",
		Manifest: "manifest",
		Created:  time.Now().UTC().Add(-49 * time.Hour),
	}
}

func testBuild(t *testing.T, opts build.Options, fn func(*build.Build, *structs.MockProvider, *exec.MockInterface, *bytes.Buffer)) {
	e := &exec.MockInterface{}
	p := &structs.MockProvider{}

	buf := &bytes.Buffer{}

	opts.Output = buf

	b, err := build.New(opts)
	require.NoError(t, err)

	b.Exec = e
	b.Provider = p

	fn(b, p, e, buf)

	e.AssertExpectations(t)
	p.AssertExpectations(t)
}
