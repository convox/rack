package appinit

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"

	yaml "gopkg.in/yaml.v2"
)

type RubyApp struct {
	af          Appfile
	environment map[string]string
	pf          Procfile
	release     Release
}

func (ra *RubyApp) GenerateEntrypoint() ([]byte, error) {
	return writeAsset("appinit/templates/entrypoint.sh", nil)
}
func (ra *RubyApp) GenerateDockerfile() ([]byte, error) {
	ra.environment["CURL_CONNECT_TIMEOUT"] = "0" // default timeouts for curl are too aggressive causing failure
	ra.environment["CURL_TIMEOUT"] = "0"
	ra.environment["STACK"] = "cedar-14"

	input := map[string]interface{}{
		"kind":        "ruby",
		"environment": ra.environment,
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
func (ra *RubyApp) GenerateManifest() ([]byte, error) {

	m := GenerateManifest(ra.pf, ra.af, ra.release)
	if len(m.Services) == 0 {
		return nil, fmt.Errorf("unable to generate manifest")
	}

	adds := []string{}
	if appFound {
		adds = append(adds, ra.af.Addons...)
	} else {
		adds = append(adds, ra.release.Addons...)
	}
	ParseAddons(adds, &m)

	return yaml.Marshal(m)
}

func (ra *RubyApp) Setup(dir string) error {
	var err error

	ra.pf, err = ReadProcfile(path.Join(dir, "Procfile"))
	if err != nil {
		return err
	}

	ra.af, err = ReadAppfile(path.Join(dir, "app.json"))
	if err != nil {
		return err
	}

	// We start a container with tailing nothing to keep it running and work inside it
	args := []string{"run", "--rm", "-d",
		"-v", fmt.Sprintf("%s:/app", dir),
		"convox/init", "tail", "-f", "/dev/null",
	}

	output, err := exec.Command(dockerBin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("buildpack contaier: %s", err)
	}

	containerID := strings.TrimSpace(string(output))

	// NOTE: The ruby-buildpack generates a release yaml file during compile
	// so we have to perform both steps and this can be time consuming, let's give feedback
	stdcli.Spinner.Prefix = "Building ruby app metadata. This could take a while... "
	stdcli.Spinner.Start()

	args = []string{"exec", containerID, "compile-release"}
	r, err := exec.Command(dockerBin, args...).CombinedOutput()
	if err != nil {
		fmt.Printf("\x08\x08FAILED\n")
		stdcli.Spinner.Stop()

		fmt.Println(string(r)) // output could be huge and not user friendly as a wall of red text if an error type
		return fmt.Errorf("buildpack compile: %s", err)
	}

	fmt.Printf("\x08\x08OK\n")
	stdcli.Spinner.Stop()

	if err := yaml.Unmarshal(r, &ra.release); err != nil {
		return err
	}

	args = []string{"exec", containerID, "profiled"}
	output, err = exec.Command(dockerBin, args...).CombinedOutput()
	if err != nil {
		fmt.Println(strings.TrimSpace(string(output)))
		return fmt.Errorf("buildpack profile: %s", err)
	}

	exec.Command(dockerBin, "rm", "--force", containerID).Run()

	ra.environment, err = parseProfiled(output)
	if err != nil {
		fmt.Errorf("parse profiled: %s", err)
	}

	// we do not want all of the buildpack's default processes on convox
	for key := range ra.release.ProcessTypes {
		if key != "web" && key != "database" {
			delete(ra.release.ProcessTypes, key)
		}
	}

	return nil
}

func parseProfiled(data []byte) (map[string]string, error) {
	env := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "export") {
			continue
		}

		l := strings.SplitN(line, "=", 2)
		if len(l) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.Replace(l[0], "export", "", -1))

		env[key] = l[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}
