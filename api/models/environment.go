package models

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/crypt"
)

type Environment map[string]string

// cleanEnvPair validates environment variable keypair format, trims spaces and surrounding single quotes.
func cleanEnvPair(value string) (string, error) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("Environment variables should be defined in key=value format. You specified: " + value)
	}

	if key := strings.TrimSpace(parts[0]); key != "" {
		val := parts[1]

		// heroku env -s adds leading and trailing single quotes to val. Strip.
		val = strings.Trim(val, "'")
		val = strings.TrimSpace(val)

		return fmt.Sprintf("%s=%s", key, val), nil
	}

	return "", fmt.Errorf("Unknown validation error")
}

// LoadEnvironment loads input into an Environment struct.
func LoadEnvironment(data []byte) (Environment, error) {
	env := Environment{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {

		value, err := cleanEnvPair(scanner.Text())
		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(value, "=", 2)
		env[parts[0]] = parts[1]
	}

	return env, nil
}

// GetEnvironment retrieves an app's current Environment.
func GetEnvironment(app string) (Environment, error) {
	a, err := GetApp(app)
	if err != nil {
		return nil, err
	}

	if a.Status == "creating" {
		return nil, fmt.Errorf("app is still being created: %s", app)
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

	env, err := LoadEnvironment(data)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// PutEnvironment creates a new release with a given Environment.
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

	err = S3Put(a.Outputs["Settings"], "env", []byte(e), false)
	if err != nil {
		return "", err
	}

	return release.Id, nil
}

// Use the Rack Settings bucket and EncryptionKey KMS key to store and retrieve
// sensitive credentials, just like app env
func GetRackSettings() (Environment, error) {
	key := os.Getenv("ENCRYPTION_KEY")
	settings := os.Getenv("SETTINGS_BUCKET")

	data, err := s3Get(settings, "env")

	if err != nil {
		// if we get a 404 from aws just return an empty environment
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			return Environment{}, nil
		}

		return nil, err
	}

	if key != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		if d, err := cr.Decrypt(key, data); err == nil {
			data = d
		}
	}

	var env Environment
	err = json.Unmarshal(data, &env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func PutRackSettings(env Environment) error {
	a, err := GetApp(os.Getenv("RACK"))

	if err != nil {
		return err
	}

	resources, err := ListResources(a.Name)

	if err != nil {
		return err
	}

	key := resources["EncryptionKey"].Id
	settings := resources["Settings"].Id

	e, err := json.Marshal(env)
	if err != nil {
		return err
	}

	if key != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		e, err = cr.Encrypt(key, e)

		if err != nil {
			return err
		}
	}

	err = S3Put(settings, "env", []byte(e), true)
	return err
}

func (e Environment) SortedNames() []string {
	names := []string{}

	for key := range e {
		names = append(names, key)
	}

	sort.Strings(names)

	return names
}

func (e Environment) Raw() string {
	lines := make([]string, len(e))

	//TODO: might make sense to quote here
	for i, name := range e.SortedNames() {
		lines[i] = fmt.Sprintf("%s=%s", name, e[name])
	}

	return strings.Join(lines, "\n")
}
