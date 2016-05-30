package main

var manifestRequired string = `www:
  environment:
    - FOO
  image: httpd
`

var manifestExplicitEqual string = `www:
  environment:
    - FOO=
  image: httpd
  command: sh
`

var manifestMapEnv string = `www:
  environment:
    FOO: bar
  image: httpd
  command: sh
`

var manifestLink string = `www:
  image: httpd
  command: sh
  links:
    - redis
redis:
  image: convox/redis
  command: sh
  ports:
    - 6379
`

// func TestStartWithMissingEnv(t *testing.T) {
//   temp, _ := ioutil.TempDir("", "convox-test")
//   appDir := temp + "/app"
//   os.Mkdir(appDir, 0777)
//   defer os.RemoveAll(appDir)

//   d1 := []byte(manifestRequired)
//   ioutil.WriteFile(appDir+"/docker-compose.yml", d1, 0777)

//   test.Runs(t,
//     test.ExecRun{
//       Command: fmt.Sprintf("convox start"),
//       Dir:     appDir,
//       Env:     map[string]string{"CONVOX_CONFIG": temp},
//       Exit:    1,
//       Stderr:  "ERROR: env expected: FOO",
//     },
//   )
// }

// func TestStartWithNoEnvOk(t *testing.T) {
//   temp, _ := ioutil.TempDir("", "convox-test")
//   appDir := temp + "/app"
//   os.Mkdir(appDir, 0777)
//   defer os.RemoveAll(appDir)

//   d1 := []byte(manifestExplicitEqual)
//   ioutil.WriteFile(appDir+"/docker-compose.yml", d1, 0777)

//   test.Runs(t,
//     test.ExecRun{
//       Command:  fmt.Sprintf("convox start"),
//       Dir:      appDir,
//       Env:      map[string]string{"CONVOX_CONFIG": temp},
//       OutMatch: "docker run",
//       Exit:     0,
//     },
//   )
// }

// func TestStartWithMapEnv(t *testing.T) {
//   temp, _ := ioutil.TempDir("", "convox-test")
//   appDir := temp + "/app"
//   os.Mkdir(appDir, 0777)
//   defer os.RemoveAll(appDir)

//   d1 := []byte(manifestMapEnv)
//   ioutil.WriteFile(appDir+"/docker-compose.yml", d1, 0777)

//   test.Runs(t,
//     test.ExecRun{
//       Command:  fmt.Sprintf("convox start"),
//       Dir:      appDir,
//       Env:      map[string]string{"CONVOX_CONFIG": temp},
//       OutMatch: "docker run",
//       Exit:     0,
//     },
//   )
// }

// func TestStartWithLink(t *testing.T) {
//   temp, _ := ioutil.TempDir("", "convox-test-link")
//   appDir := temp + "/app"
//   os.Mkdir(appDir, 0777)
//   defer os.RemoveAll(appDir)

//   d1 := []byte(manifestLink)
//   ioutil.WriteFile(appDir+"/docker-compose.yml", d1, 0777)

//   test.Runs(t,
//     test.ExecRun{
//       Command:  fmt.Sprintf("convox start"),
//       Dir:      appDir,
//       Env:      map[string]string{"CONVOX_CONFIG": temp},
//       OutMatch: "REDIS_URL",
//       Exit:     0,
//     },
//   )
// }
