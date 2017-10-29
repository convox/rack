package main

import (
	"testing"

	"github.com/convox/rack/client"
	"github.com/convox/rack/test"
)

func TestDeployPreventAgainstNonRunningStatus(t *testing.T) {
	ts := testServer(t,
		test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "creating"}},
	)

	defer ts.Close()

	test.Runs(t,
		test.ExecRun{
			Command: "convox deploy --app foo",
			Exit:    1,
			Stdout:  "",
			Stderr:  "ERROR: unable to deploy foo in a non-running status: creating\n",
		},
	)
}

// func TestDeployNoManifestFound(t *testing.T) {
//   ts := testServer(t,
//     test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
//   )

//   defer ts.Close()

//   test.Runs(t,
//     test.ExecRun{
//       Command: "convox deploy --app foo",
//       Exit:    1,
//       Stderr:  "ERROR: no docker-compose.yml found, try `convox init` to generate one\n",
//     },
//   )
// }

// func TestDeployInvalidCronInManifest(t *testing.T) {
//   ts := testServer(t,
//     test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
//   )

//   defer ts.Close()

//   test.Runs(t,
//     test.ExecRun{
//       Command: "convox deploy --file docker-compose.invalid-cron-label.yml --app foo",
//       Dir:     "../../manifest/fixtures",
//       Exit:    1,
//       Stderr:  "ERROR: Cron task my_job is not valid (cron names can contain only alphanumeric characters, dashes and must be between 4 and 30 characters)\n",
//     },
//   )
// }

// func TestDeployDuplicateCronInManifest(t *testing.T) {
//   ts := testServer(t,
//     test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
//   )

//   defer ts.Close()

//   test.Runs(t,
//     test.ExecRun{
//       Command: "convox deploy --file docker-compose.duplicate-cron-label.yml --app foo",
//       Dir:     "../../manifest/fixtures",
//       Exit:    1,
//       Stderr:  "ERROR: invalid docker-compose.duplicate-cron-label.yml: error loading manifest: duplicate cron label convox.cron.myjob\n",
//     },
//   )
// }
