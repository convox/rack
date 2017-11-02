package main

/* HELP CHECKS */
// http://www.gnu.org/prep/standards/html_node/_002d_002dhelp.html

// Note: when --help precedes a subcommand, it shows Convox help, not the subcommand help
// This is a known issue: https://github.com/urfave/cli/issues/318

var convoxUsage = `convox: command-line application management

Usage:
  convox <command> [subcommand] [options...] [args...]

Commands: (convox <command> --help)
  api                  make a rest api call to a convox endpoint
  apps                 list deployed apps
  build                create a new build
  builds               manage an app's builds
  certs                list certificates
  deploy               deploy an app to AWS
  doctor               check your app for common Convox compatibility issues
  env                  manage an app's environment variables
  exec                 exec a command in a process in your Convox rack
  init                 initialize an app for local development
  install              install convox into an aws account
  instances            list your Convox rack's instances
  login                log into your convox rack
  logs                 stream the logs for an application
  proxy                proxy local ports into a rack
  ps                   list an app's processes
  rack                 manage your Convox rack
  racks                list your Convox racks
  registries           manage private registries
  releases             list an app's releases
  run                  run a one-off command in your Convox rack
  scale                scale an app's processes
  resources, services  manage external resources [prev. services]
  ssl                  manage ssl certificates
  start                start an app for local development
  switch               switch to another Convox rack
  uninstall            uninstall a convox rack
  update               update the cli
  help, h              
  
Options:
  --app value, -a value  app name inferred from current directory if not specified
  --rack value           rack name
  --help, -h             show help
  --version, -v          print the version
  `

// func TestHelpFlag(t *testing.T) {
//   assert.Equal(t, stdcli.HelpFlags, []string{"--help", "-h", "h", "help"})

//   ts := testServer(t,
//     test.Http{
//       Method: "GET",
//       Path:   "/",
//       Code:   200,
//     },
//   )

//   defer ts.Close()
//   for _, hf := range stdcli.HelpFlags {
//     c := fmt.Sprintf("convox %s", hf)
//     test.Runs(t,
//       test.ExecRun{
//         Command: c,
//         Exit:    0,
//         Stdout:  convoxUsage,
//       },
//     )
//   }
// }

// func TestWaitFlag(t *testing.T) {
//   wf := waitFlag
//   require.IsType(t, cli.BoolFlag{}, wf)
//   assert.Equal(t, "CONVOX_WAIT", wf.EnvVar)
//   assert.Equal(t, "wait", wf.Name)
// }
