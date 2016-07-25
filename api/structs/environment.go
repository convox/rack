package structs

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Environment is of type map used to store environment variables.
type Environment map[string]string

// LoadEnvironment sets the environment from data.
func (e Environment) LoadEnvironment(data []byte) Environment {

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)

		if len(parts) == 2 {
			if key := strings.TrimSpace(parts[0]); key != "" {
				e[key] = parts[1]
			}
		}
	}

	return e
}

// SortedNames returns a slice of environment variables sorted by name.
func (e Environment) SortedNames() []string {
	names := []string{}

	for key := range e {
		names = append(names, key)
	}

	sort.Strings(names)

	return names
}

// Raw returns the environment variables as one string separated by a newline.
func (e Environment) Raw() string {
	lines := make([]string, len(e))

	//TODO: might make sense to quote here
	for i, name := range e.SortedNames() {
		lines[i] = fmt.Sprintf("%s=%s", name, e[name])
	}

	return strings.Join(lines, "\n")
}

// LoadRaw reads a raw string (key/values separated by a newline) to load environment variables
func (e Environment) LoadRaw(raw string) {
	if raw == "" {
		return
	}

	for _, rawKV := range strings.Split(raw, "\n") {
		keyValue := strings.SplitN(rawKV, "=", 2)
		e[keyValue[0]] = keyValue[1]
	}
}

// List retuns a string slic of environment variables. e.g ["KEY=VALUE"]
func (e Environment) List() []string {

	list := []string{}

	for key, value := range e {
		list = append(list, fmt.Sprintf("%s=%s", key, value))
	}

	return list
}
