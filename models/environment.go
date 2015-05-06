package models

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type Environment map[string]string

func GetEnvironment(app string) (Environment, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	data, err := s3Get(a.Outputs["Settings"], "env")

	if err != nil {
		return nil, err
	}

	env := Environment{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)

		if len(parts) == 2 {
			if key := strings.TrimSpace(parts[0]); key != "" {
				env[key] = parts[1]
			}
		}
	}

	return env, nil
}

func PutEnvironment(app string, env Environment) error {
	a, err := GetApp(app)

	if err != nil {
		return err
	}

	lines := make([]string, len(env))

	for i, name := range env.SortedNames() {
		lines[i] = fmt.Sprintf("%s=%s", name, env[name])
	}

	data := []byte(strings.Join(lines, "\n"))

	return s3Put(a.Outputs["Settings"], "env", data, true)
}

func (e Environment) SortedNames() []string {
	names := []string{}

	for key, _ := range e {
		names = append(names, key)
	}

	sort.Strings(names)

	return names
}
