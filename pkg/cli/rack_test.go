package cli_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider"
	"github.com/convox/stdcli"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRack(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name      name",
			"Provider  provider",
			"Region    region",
			"Router    domain",
			"Status    running",
			"Version   21000101000000",
		})
	})
}

func TestRackError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackInternal(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemInternal(), nil)

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name      name",
			"Provider  provider",
			"Region    region",
			"Router    domain (external)",
			"          domain-internal (internal)",
			"Status    running",
			"Version   20180901000000",
		})
	})
}

func TestRackNLB(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemNLB(), nil)

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name      name",
			"Provider  provider",
			"Region    region",
			"Router    domain",
			"NLB       nlb-host.elb.region.amazonaws.com (203.0.113.10, 203.0.113.11, 203.0.113.12)",
			"Status    running",
			"Version   21000101000000",
		})
	})
}

func TestRackNLBInternal(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemNLBInternal(), nil)

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name          name",
			"Provider      provider",
			"Region        region",
			"Router        domain",
			"NLB Internal  nlb-internal-host.elb.region.amazonaws.com",
			"Status        running",
			"Version       21000101000000",
		})
	})
}

func TestRackInstall(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/auth", r.URL.Path)
			user, pass, _ := r.BasicAuth()
			require.Equal(t, "convox", user)
			require.Equal(t, "password", pass)
		}))

		tsu, err := url.Parse(ts.URL)
		require.NoError(t, err)

		opts := structs.SystemInstallOptions{
			Name:       options.String("foo"),
			Parameters: map[string]string{},
			Version:    options.String("bar"),
		}
		provider.Mock.On("SystemInstall", mock.Anything, opts).Once().Return(fmt.Sprintf("https://convox:password@%s", tsu.Host), nil).Run(func(args mock.Arguments) {
			w := args.Get(0).(io.Writer)
			fmt.Fprintf(w, "line1\n")
			fmt.Fprintf(w, "line2\n")
		})

		res, err := testExecute(e, "rack install test -n foo -v bar", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"line1",
			"line2",
		})

		data, err := os.ReadFile(filepath.Join(e.Settings, "auth"))
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("{\n  \"%s\": \"password\"\n}", tsu.Host), string(data))

		data, err = os.ReadFile(filepath.Join(e.Settings, "host"))
		require.NoError(t, err)
		require.Equal(t, tsu.Host, string(data))
	})
}

func TestRackInstallError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.SystemInstallOptions{
			Name:       options.String("foo"),
			Parameters: map[string]string{},
			Version:    options.String("bar"),
		}
		provider.Mock.On("SystemInstall", mock.Anything, opts).Return("", fmt.Errorf("err1"))

		res, err := testExecute(e, "rack install test -n foo -v bar", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackLogs(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemLogs", structs.LogsOptions{Prefix: options.Bool(true)}).Return(testLogs(fxLogs()), nil)

		res, err := testExecute(e, "rack logs", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			fxLogs()[0],
			fxLogs()[1],
		})
	})
}

func TestRackLogsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemLogs", structs.LogsOptions{Prefix: options.Bool(true)}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack logs", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackParams(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)

		res, err := testExecute(e, "rack params", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Autoscale   Yes",
			"ParamFoo    value1",
			"ParamOther  value2",
		})
	})
}

func TestRackParamsError(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack params", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackParamsSet(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.SystemUpdateOptions{
			Parameters: map[string]string{
				"Foo": "bar",
				"Baz": "qux",
			},
		}
		i.On("SystemUpdate", opts).Return(nil)

		res, err := testExecute(e, "rack params set Foo=bar Baz=qux", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating parameters... OK"})
	})
}

func TestRackParamsSetError(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.SystemUpdateOptions{
			Parameters: map[string]string{
				"Foo": "bar",
				"Baz": "qux",
			},
		}
		i.On("SystemUpdate", opts).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "rack params set Foo=bar Baz=qux", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Updating parameters... "})
	})
}

func TestRackParamsSetClassic(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("AppParametersSet", "name", map[string]string{"Foo": "bar", "Baz": "qux"}).Return(nil)

		res, err := testExecute(e, "rack params set Foo=bar Baz=qux", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating parameters... OK"})
	})
}

func TestRackParamsSetClassicError(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("AppParametersSet", "name", map[string]string{"Foo": "bar", "Baz": "qux"}).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "rack params set Foo=bar Baz=qux", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Updating parameters... "})
	})
}

// --- Integration tests for `rack params` masking and -g/--reveal flags ---

// fxSystemWithSensitive is a system fixture with values for masked params.
func fxSystemWithSensitive() *structs.System {
	return &structs.System{
		Parameters: map[string]string{
			"Password":     "secret123",
			"HttpProxy":    "http://user:pass@proxy.corp:8080",
			"VPCCIDR":      "10.0.0.0/16",
			"Version":      "3.25.0",
			"Autoscale":    "Yes",
			"InstanceType": "t3.medium",
			"Telemetry":    "Yes",
		},
	}
}

func TestRackParamsMaskOnTTY(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return true }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemWithSensitive(), nil)

		res, err := testExecute(e, "rack params", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"Autoscale     Yes",
			"HttpProxy     **********",
			"InstanceType  t3.medium",
			"Password      **********",
			"Telemetry     Yes",
			"VPCCIDR       10.0.0.0/16",
			"Version       3.25.0",
		})
	})
}

func TestRackParamsNoMaskOnPipe(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemWithSensitive(), nil)

		res, err := testExecute(e, "rack params", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"Autoscale     Yes",
			"HttpProxy     http://user:pass@proxy.corp:8080",
			"InstanceType  t3.medium",
			"Password      secret123",
			"Telemetry     Yes",
			"VPCCIDR       10.0.0.0/16",
			"Version       3.25.0",
		})
	})
}

func TestRackParamsRevealBypassesMaskOnTTY(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return true }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemWithSensitive(), nil)

		res, err := testExecute(e, "rack params --reveal", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"Autoscale     Yes",
			"HttpProxy     http://user:pass@proxy.corp:8080",
			"InstanceType  t3.medium",
			"Password      secret123",
			"Telemetry     Yes",
			"VPCCIDR       10.0.0.0/16",
			"Version       3.25.0",
		})
	})
}

// TestRackParamsMaskedTTYWithGroupFilter verifies masking applies inside a
// group filter on TTY. Password is dual-listed in the security group, so
// `-g security` on TTY must still mask Password.
func TestRackParamsMaskedTTYWithGroupFilter(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return true }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		sys := &structs.System{
			Parameters: map[string]string{
				"Password":   "secret",
				"Encryption": "Yes",
				"WhiteList":  "1.2.3.4/32",
				"VPCCIDR":    "10.0.0.0/16",
				"Autoscale":  "Yes",
			},
		}
		i.On("SystemGet").Return(sys, nil)

		res, err := testExecute(e, "rack params -g security", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"Encryption  Yes",
			"Password    **********",
			"WhiteList   1.2.3.4/32",
		})
	})
}

// TestRackParamsSensitiveEmptyValueNotMasked verifies the `v != ""` guard:
// a sensitive param with an empty string value stays empty, not "**********".
func TestRackParamsSensitiveEmptyValueNotMasked(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return true }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		sys := &structs.System{
			Parameters: map[string]string{
				"Password":  "",
				"HttpProxy": "",
				"VPCCIDR":   "10.0.0.0/16",
				"Autoscale": "Yes",
			},
		}
		i.On("SystemGet").Return(sys, nil)

		res, err := testExecute(e, "rack params", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.NotContains(t, res.Stdout, "**********", "empty sensitive values must not render as asterisks")
		require.Contains(t, res.Stdout, "Password")
		require.Contains(t, res.Stdout, "HttpProxy")
	})
}

func TestRackParamsGroupFilterSecurity(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		sys := &structs.System{
			Parameters: map[string]string{
				"Password":   "secret",
				"VPCCIDR":    "10.0.0.0/16",
				"Autoscale":  "Yes",
				"Encryption": "Yes",
				"WhiteList":  "1.2.3.4/32",
			},
		}
		i.On("SystemGet").Return(sys, nil)

		res, err := testExecute(e, "rack params -g security", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"Encryption  Yes",
			"Password    secret",
			"WhiteList   1.2.3.4/32",
		})
	})
}

// TestRackParamsGroupFilterEmptyFallthrough: -g "" falls through to full
// unfiltered dump per V3 parity (caller-side guard prevents resolveGroup
// from being called with empty input).
func TestRackParamsGroupFilterEmptyFallthrough(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		sys := &structs.System{
			Parameters: map[string]string{
				"Password":  "secret",
				"VPCCIDR":   "10.0.0.0/16",
				"Autoscale": "Yes",
			},
		}
		i.On("SystemGet").Return(sys, nil)

		res, err := testExecute(e, "rack params -g ''", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"Autoscale  Yes",
			"Password   secret",
			"VPCCIDR    10.0.0.0/16",
		})
	})
}

// TestRackParamsGroupFilterWhitespaceOnlyErrors: -g "   " passes the caller-
// side non-empty guard, reaches resolveGroup, which trims to empty and
// returns the `group name required` error. Exit non-zero.
func TestRackParamsGroupFilterWhitespaceOnlyErrors(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		res, err := testExecute(e, "rack params -g '   '", nil)
		require.NoError(t, err)
		require.NotEqual(t, 0, res.Code)
		require.Contains(t, res.Stderr, "group name required")
		require.Contains(t, res.Stderr, "available groups")
	})
}

func TestRackParamsGroupFilterUnknownErrors(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		res, err := testExecute(e, "rack params -g notarealgroup", nil)
		require.NoError(t, err)
		require.NotEqual(t, 0, res.Code)
		require.Contains(t, res.Stderr, "not found")
		require.Contains(t, res.Stderr, "network")
	})
}

// TestRackParamsEmptyGroupNotice: group resolves but rack has zero matching
// params — print stderr NOTICE + exit 0.
func TestRackParamsEmptyGroupNotice(t *testing.T) {
	prev := cli.IsTerminalFn
	cli.IsTerminalFn = func(_ *stdcli.Context) bool { return false }
	t.Cleanup(func() { cli.IsTerminalFn = prev })

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		sys := &structs.System{
			Parameters: map[string]string{
				"Autoscale": "Yes",
				"VPCCIDR":   "10.0.0.0/16",
			},
		}
		i.On("SystemGet").Return(sys, nil)

		res, err := testExecute(e, "rack params -g nlb", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Contains(t, res.Stderr, "NOTICE: no params in group 'nlb' for this rack")
	})
}

func TestRackPs(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemProcesses", structs.SystemProcessesOptions{}).Return(structs.Processes{*fxProcess(), *fxProcessPending()}, nil)

		res, err := testExecute(e, "rack ps", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID    APP   SERVICE  STATUS   RELEASE   STARTED     COMMAND",
			"pid1  app1  name     running  release1  2 days ago  command",
			"pid1  app1  name     pending  release1  2 days ago  command",
		})
	})
}

func TestRackPsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemProcesses", structs.SystemProcessesOptions{}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack ps", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackPsAll(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemProcesses", structs.SystemProcessesOptions{All: options.Bool(true)}).Return(structs.Processes{*fxProcess(), *fxProcessPending()}, nil)

		res, err := testExecute(e, "rack ps -a", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID    APP   SERVICE  STATUS   RELEASE   STARTED     COMMAND",
			"pid1  app1  name     running  release1  2 days ago  command",
			"pid1  app1  name     pending  release1  2 days ago  command",
		})
	})
}

func TestRackReleases(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemReleases").Return(structs.Releases{*fxRelease(), *fxRelease()}, nil)

		res, err := testExecute(e, "rack releases", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"VERSION   UPDATED",
			"release1  2 days ago",
			"release1  2 days ago",
		})
	})
}

func TestRackReleasesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemReleases").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack releases", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackScale(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)

		res, err := testExecute(e, "rack scale", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Autoscale  Yes",
			"Count      1",
			"Status     running",
			"Type       type",
		})
	})
}

func TestRackScaleError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack scale", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackScaleUpdate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemUpdate", structs.SystemUpdateOptions{Count: options.Int(5), Type: options.String("type1")}).Return(nil)

		res, err := testExecute(e, "rack scale -c 5 -t type1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Scaling rack... OK"})
	})
}

func TestRackScaleUpdateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("SystemUpdate", structs.SystemUpdateOptions{Count: options.Int(5), Type: options.String("type1")}).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "rack scale -c 5 -t type1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Scaling rack... "})
	})
}

func TestRackUninstall(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.SystemUninstallOptions{
			Force: options.Bool(true),
		}
		provider.Mock.On("SystemUninstall", "foo", mock.Anything, opts).Once().Return(nil).Run(func(args mock.Arguments) {
			w := args.Get(1).(io.Writer)
			fmt.Fprintf(w, "line1\n")
			fmt.Fprintf(w, "line2\n")
		})

		res, err := testExecute(e, "rack uninstall test foo --force", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"line1",
			"line2",
		})
	})
}

func TestRackUninstallError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.SystemUninstallOptions{
			Force: options.Bool(true),
		}
		provider.Mock.On("SystemUninstall", "foo", mock.Anything, opts).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "rack uninstall test foo --force", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackUninstallWithoutForce(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		res, err := testExecute(e, "rack uninstall test foo", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: must use --force for non-interactive uninstall"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRackUpdate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemUpdate", structs.SystemUpdateOptions{Version: options.String("version1")}).Return(nil)

		res, err := testExecute(e, "rack update version1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating to version1... OK"})
	})
}

func TestRackUpdateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemUpdate", structs.SystemUpdateOptions{Version: options.String("version1")}).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "rack update version1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Updating to version1... "})
	})
}

func TestRackWait(t *testing.T) {
	testClientWait(t, 100*time.Millisecond, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{
			Prefix: options.Bool(true),
			Since:  options.Duration(5 * time.Second),
		}
		i.On("SystemLogs", opts).Return(testLogs(fxLogsSystem()), nil).Once()
		i.On("SystemGet").Return(&structs.System{Status: "updating"}, nil).Twice()
		i.On("SystemGet").Return(fxSystem(), nil)

		res, err := testExecute(e, "rack wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Waiting for rack... ",
			fxLogsSystem()[0],
			fxLogsSystem()[1],
			"OK",
		})
	})
}

func TestRackWaitError(t *testing.T) {
	testClientWait(t, 100*time.Millisecond, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{
			Prefix: options.Bool(true),
			Since:  options.Duration(5 * time.Second),
		}
		i.On("SystemLogs", opts).Return(testLogs(fxLogsSystem()), nil).Once()
		i.On("SystemGet").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "rack wait", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			"Waiting for rack... ",
			fxLogsSystem()[0],
			fxLogsSystem()[1],
		})
	})
}
