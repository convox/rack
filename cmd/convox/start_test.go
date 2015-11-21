package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/convox/rack/test"
)

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

func TestStartWithMissingEnv(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")
	appDir := temp + "/app"
	os.Mkdir(appDir, 0777)
	defer os.RemoveAll(appDir)

	d1 := []byte(manifestRequired)
	ioutil.WriteFile(appDir+"/docker-compose.yml", d1, 0777)

	test.Runs(t,
		test.ExecRun{
			Command: fmt.Sprintf("convox start"),
			Dir:     appDir,
			Env:     map[string]string{"CONVOX_CONFIG": temp},
			Exit:    1,
			Stderr:  "ERROR: env expected: FOO",
		},
	)
}

func TestStartWithNoEnvOk(t *testing.T) {
	temp, _ := ioutil.TempDir("", "convox-test")
	appDir := temp + "/app"
	os.Mkdir(appDir, 0777)
	defer os.RemoveAll(appDir)

	d1 := []byte(manifestExplicitEqual)
	ioutil.WriteFile(appDir+"/docker-compose.yml", d1, 0777)

	test.Runs(t,
		test.ExecRun{
			Command:  fmt.Sprintf("convox start"),
			Dir:      appDir,
			Env:      map[string]string{"CONVOX_CONFIG": temp},
			OutMatch: "docker run",
			Exit:     0,
		},
	)
}
