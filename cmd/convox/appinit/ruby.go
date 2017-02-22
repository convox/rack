package appinit

import (
	"fmt"
	"os/exec"
	"path"

	yaml "gopkg.in/yaml.v2"

	"github.com/convox/rack/cmd/convox/stdcli"
)

type RubyApp struct{}

func (ra *RubyApp) GenerateEntrypoint() ([]byte, error) {
	return writeAsset("appinit/templates/entrypoint.sh", nil)
}
func (ra *RubyApp) GenerateDockerfile() ([]byte, error) {
	input := map[string]interface{}{
		"kind": "ruby",
		"environment": map[string]string{
			"CURL_CONNECT_TIMEOUT": "0", // default timeouts for curl are too aggressive causing failure
			"CURL_TIMEOUT":         "0",
			"STACK":                "cedar-14",
		},
	}
	return writeAsset("appinit/templates/Dockerfile", input)
}
func (ra *RubyApp) GenerateDockerIgnore() ([]byte, error) {
	input := map[string]interface{}{
		"ignoreFiles": []string{
			"./tmp",
		},
	}
	return writeAsset("appinit/templates/dockerignore", input)
}
func (ra *RubyApp) GenerateManifest(dir string) ([]byte, error) {
	pf, err := ReadProcfile(path.Join(dir, "Procfile"))
	if err != nil {
		return nil, err
	}

	am, err := ReadAppfile(path.Join(dir, "app.json"))
	if err != nil {
		return nil, err
	}

	var release Release
	if len(pf) == 0 || !appFound {
		var r []byte
		args := []string{"run", "--rm", "-v", fmt.Sprintf("%s:/tmp/app", dir), "convox/init"}

		// NOTE: The ruby-buildpack generates a yaml file during compile so we have to perform both steps
		// this can be time consuming, let's give feedback
		stdcli.Spinner.Prefix = "Building ruby app metadata. This could take a while... "
		stdcli.Spinner.Start()

		r, err := exec.Command(dockerBin, append(args, "compile-release")...).CombinedOutput()
		if err != nil {
			fmt.Printf("\x08\x08FAILED\n")
			stdcli.Spinner.Stop()

			fmt.Println(string(r)) // output could be huge and not user friendly as a wall of red text if an error type
			return nil, fmt.Errorf("unable to complie ruby app")
		}

		fmt.Printf("\x08\x08OK\n")
		stdcli.Spinner.Stop()

		if err := yaml.Unmarshal(r, &release); err != nil {
			return nil, err
		}

		// we do not want all of the buildpack's default processes on convox
		for key := range release.ProcessTypes {
			if key != "web" && key != "database" {
				delete(release.ProcessTypes, key)

			}
		}

	}

	m := GenerateManifest(pf, am, release)
	if len(m.Services) == 0 {
		return nil, fmt.Errorf("unable to generate manifest")
	}

	adds := []string{}
	if appFound {
		adds = append(adds, am.Addons...)
	} else {
		adds = append(adds, release.Addons...)
	}
	ParseAddons(adds, &m)

	return yaml.Marshal(m)
}
