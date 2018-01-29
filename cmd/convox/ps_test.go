package main

import (
	"testing"
	"time"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestPs(t *testing.T) {
	tr := testServer(t,
		test.Http{
			Method: "GET",
			Path:   "/apps/myapp/processes",
			Code:   200,
			Response: client.Processes{
				client.Process{
					Id:      "fooID",
					App:     "fooApp",
					Command: "fooCommand",
					Host:    "fooHost",
					Image:   "fooImage",
					Name:    "fooName",
					Ports:   []string{"fooPorts"},
					Release: "fooRelease",
					Cpu:     256,
					Memory:  256,
					Started: time.Now(),
				},
			},
		},
	)

	defer tr.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox ps --app myapp",
			Exit:    0,
			Stdout:  "ID     NAME     RELEASE     STARTED  COMMAND\nfooID  fooName  fooRelease  now      fooCommand\n",
		},
	)
}

/* HELP-USAGE CHECKS */

var psUsageWithoutHelpFlag = `convox ps: list an app's processes`

var psUsage = `convox ps: list an app's processes`

var psInfoUsage = `convox ps info: show info for a process`

var psStopUsage = `convox ps stop: stop a process by its id`

var psMissingProcessID = `ERROR: 1 argument is required: <process id>`

// func TestPsHelpFlag(t *testing.T) {
//   tests := []test.ExecRun{
//     test.ExecRun{
//       Command:  "convox ps h",
//       OutMatch: psUsage,
//       Env:      DebuglessEnv,
//     },
//     test.ExecRun{
//       Command:  "convox ps help",
//       OutMatch: psUsage,
//       Env:      DebuglessEnv,
//     },
//     test.ExecRun{
//       Command:  "convox ps -h",
//       OutMatch: psUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps --help",
//       OutMatch: psUsage,
//     },
//     test.ExecRun{
//       Command:  "convox h ps",
//       OutMatch: psUsageWithoutHelpFlag,
//     },
//     test.ExecRun{
//       Command:  "convox help ps",
//       OutMatch: psUsageWithoutHelpFlag,
//     },

//     // ps stop
//     test.ExecRun{
//       Command:  "convox ps stop",
//       Exit:     129,
//       Stderr:   psMissingProcessID,
//       OutMatch: psStopUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps stop -h",
//       OutMatch: psStopUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps stop --help",
//       OutMatch: psStopUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps stop h",
//       OutMatch: psStopUsage,
//       Env:      DebuglessEnv,
//     },
//     test.ExecRun{
//       Command:  "convox ps stop help",
//       OutMatch: psStopUsage,
//       Env:      DebuglessEnv,
//     },
//     test.ExecRun{
//       Command:  "convox ps h stop",
//       OutMatch: psStopUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps help stop",
//       OutMatch: psStopUsage,
//       Env:      DebuglessEnv,
//     },

//     // ps info
//     test.ExecRun{
//       Command:  "convox ps info",
//       Exit:     129,
//       Stderr:   psMissingProcessID,
//       OutMatch: psInfoUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps info -h",
//       OutMatch: psInfoUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps info --help",
//       OutMatch: psInfoUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps -h info",
//       OutMatch: psUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps --help info",
//       OutMatch: psUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps h info",
//       OutMatch: psInfoUsage,
//     },
//     test.ExecRun{
//       Command:  "convox ps help info",
//       OutMatch: psInfoUsage,
//     },
//   }

//   assert.Equal(t, stdcli.HelpFlags, []string{"--help", "-h", "h", "help"})

//   ts := testServer(t,
//     test.Http{
//       Method:   "GET",
//       Path:     "/apps/myapp/processes",
//       Code:     200,
//       Response: "bar",
//       Headers:  map[string]string{"Rack": "myorg/staging"},
//     },
//   )
//   defer ts.Close()

//   for _, myTest := range tests {
//     test.Runs(t, myTest)
//   }

// }

func TestPsInfoMissingArg(t *testing.T) {
	ts := testServer(t,
		test.Http{
			Method:   "GET",
			Path:     "/apps/myapp/processes",
			Code:     200,
			Response: "bar",
			Headers:  map[string]string{"Rack": "myorg/staging"},
		},
	)
	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command:  "convox ps info",
			Env:      DebugfulEnv,
			Exit:     129,
			OutMatch: psInfoUsage,
			Stderr:   psMissingProcessID,
		},
		test.ExecRun{
			Command:  "convox ps info",
			Env:      DebuglessEnv,
			Exit:     129,
			OutMatch: psInfoUsage,
		},
	)
}
