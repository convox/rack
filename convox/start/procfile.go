package start

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var procfileEntryRegexp = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func Procfile(base string) error {
	app := filepath.Base(base)
	image := fmt.Sprintf("%s-app", app)

	dockerfile := filepath.Join(base, "Dockerfile")

	err := ioutil.WriteFile(dockerfile, []byte("FROM convox/cedar"), 0644)

	if err != nil {
		return err
	}

	err = run("docker", "build", "-f", dockerfile, "-t", image, base)

	if err != nil {
		return err
	}

	err = os.Remove(dockerfile)

	if err != nil {
		return err
	}

	data, err := query("docker", "inspect", "-f", "{{ json .ContainerConfig.ExposedPorts }}", image)

	if err != nil {
		return err
	}

	var ports map[string]interface{}

	err = json.Unmarshal(data, &ports)

	if err != nil {
		return err
	}

	args := []string{"run"}
	cur := 5000

	for port, _ := range ports {
		args = append(args, "-p")
		args = append(args, fmt.Sprintf("%d:%s", cur, strings.Split(port, "/")[0]))
		cur += 1
	}

	args = append(args, image)

	data, err = ioutil.ReadFile(filepath.Join(base, "Procfile"))

	if err != nil {
		return err
	}

	procfile, err := parseProcfile(data)

	if err != nil {
		return err
	}

	args = append(args, procfile["web"])

	err = run("docker", args...)

	if err != nil {
		return err
	}

	return nil
}

func parseProcfile(data []byte) (map[string]string, error) {
	pf := map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := procfileEntryRegexp.FindStringSubmatch(scanner.Text())

		if len(parts) > 0 {
			pf[parts[1]] = parts[2]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Reading Procfile: %s", err)
	}

	return pf, nil
}
