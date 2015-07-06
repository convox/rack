package build

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

var procfileEntryRegexp = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func Procfile(base string, app string) error {
	data, err := ioutil.ReadFile(filepath.Join(base, "Procfile"))

	if err != nil {
		return err
	}

	procfile, err := parseProcfile(data)

	if err != nil {
		return err
	}

	data, err = ManifestFromProcfile(procfile)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "docker-compose.yml"), data, 0644)

	if err != nil {
		return err
	}

	data, err = genDockerfile(procfile)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "Dockerfile"), data, 0644)

	if err != nil {
		return err
	}

	return nil
}

func genDockerfile(procs map[string]string) ([]byte, error) {
	return []byte(`FROM convox/cedar`), nil
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
