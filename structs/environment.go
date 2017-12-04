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

func (e Environment) Load(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)

		if len(parts) == 2 {
			if key := strings.TrimSpace(parts[0]); key != "" {
				e[key] = parts[1]
			}
		}
	}

	return scanner.Err()
}

func (e Environment) String() string {
	lines := []string{}

	for k, v := range e {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(lines)

	return strings.Join(lines, "\n")
}

// SortedNames returns a slice of environment variables sorted by name.
// func (e Environment) SortedNames() []string {
//   names := []string{}

//   for key := range e {
//     names = append(names, key)
//   }

//   sort.Strings(names)

//   return names
// }

// // Raw returns the environment variables as one string separated by a newline.
// func (e Environment) Raw() string {
//   lines := make([]string, len(e))

//   //TODO: might make sense to quote here
//   for i, name := range e.SortedNames() {
//     lines[i] = fmt.Sprintf("%s=%s", name, e[name])
//   }

//   return strings.Join(lines, "\n")
// }

// // List retuns a string slic of environment variables. e.g ["KEY=VALUE"]
// func (e Environment) List() []string {

//   list := []string{}

//   for key, value := range e {
//     list = append(list, fmt.Sprintf("%s=%s", key, value))
//   }

//   return list
// }
