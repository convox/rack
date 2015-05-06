package models

import (
	"bufio"
	"bytes"
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
			env[parts[0]] = parts[1]
		}
	}

	return env, nil
}

func (e Environment) SortedNames() []string {
	names := []string{}

	for key, _ := range e {
		names = append(names, key)
	}

	sort.Strings(names)

	return names
}
