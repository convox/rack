package models

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
)

type Environment map[string]string

func LoadEnvironment(data []byte) Environment {
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

	return env
}

func GetEnvironment(app string) (Environment, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	data, err := s3Get(a.Outputs["Settings"], "env")

	if err != nil {

		// if we get a 404 from aws just return an empty environment
		if awsError, ok := err.(aws.APIError); ok && awsError.StatusCode == 404 {
			return Environment{}, nil
		}

		return nil, err
	}

	return LoadEnvironment(data), nil
}

func PutEnvironment(app string, env Environment) error {
	a, err := GetApp(app)

	if err != nil {
		return err
	}

	return s3Put(a.Outputs["Settings"], "env", []byte(env.Raw()), true)
}

func (e Environment) SortedNames() []string {
	names := []string{}

	for key, _ := range e {
		names = append(names, key)
	}

	sort.Strings(names)

	return names
}

func (e Environment) Raw() string {
	lines := make([]string, len(e))

	for i, name := range e.SortedNames() {
		lines[i] = fmt.Sprintf("%s=%s", name, e[name])
	}

	return strings.Join(lines, "\n")
}
