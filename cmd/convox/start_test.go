package main

// func TestStartInvalidManifest(t *testing.T) {
//   ts := testServer(t,
//     test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
//   )

//   defer ts.Close()

//   test.Runs(t,
//     test.ExecRun{
//       Command: "convox start --file docker-compose.invalid-cron-label.yml",
//       Dir:     "../../manifest/fixtures",
//       Exit:    1,
//       Stderr:  "ERROR: Cron task my_job is not valid (cron names can contain only alphanumeric characters, dashes and must be between 4 and 30 characters)\n",
//     },
//   )
// }
// func TestStartDuplicateCronInManifest(t *testing.T) {
//   ts := testServer(t,
//     test.Http{Method: "GET", Path: "/apps/foo", Code: 200, Response: client.App{Name: "foo", Status: "running"}},
//   )

//   defer ts.Close()

//   test.Runs(t,
//     test.ExecRun{
//       Command: "convox start --file docker-compose.duplicate-cron-label.yml",
//       Dir:     "../../manifest/fixtures",
//       Exit:    1,
//       Stderr:  "ERROR: error loading manifest: duplicate cron label convox.cron.myjob\n",
//     },
//   )
// }
