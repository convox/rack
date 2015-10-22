package models

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/crypt"
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
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			return Environment{}, nil
		}

		return nil, err
	}

	if a.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		if d, err := cr.Decrypt(a.Parameters["Key"], data); err == nil {
			data = d
		}
	}

	return LoadEnvironment(data), nil
}

func PutEnvironment(app string, env Environment) (string, error) {
	a, err := GetApp(app)

	if err != nil {
		return "", err
	}

	switch a.Status {
	case "creating":
		return "", fmt.Errorf("app is still creating: %s", app)
	case "running", "updating":
	default:
		return "", fmt.Errorf("unable to set environment on app: %s", app)
	}

	release, err := a.ForkRelease()

	if err != nil {
		return "", err
	}

	release.Env = env.Raw()

	err = release.Save()

	if err != nil {
		return "", err
	}

	e := []byte(env.Raw())

	if a.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		e, err = cr.Encrypt(a.Parameters["Key"], e)

		if err != nil {
			return "", err
		}
	}

	err = S3Put(a.Outputs["Settings"], "env", []byte(e), true)

	if err != nil {
		return "", err
	}

	NotifySuccess("release:create", map[string]string{"id": release.Id})
	return release.Id, nil
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
