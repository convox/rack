package models

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/crypt"
)

type Environment map[string]string

// LoadEnvironment loads input into an Environment struct.
func LoadEnvironment(data []byte) (Environment, error) {
	env := Environment{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {

		key, value, err := ParseEnvLine(scanner.Text())
		if err != nil {
			return nil, err
		}

		if key != "" {
			env[key] = value
		}
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

	err = S3Put(a.Outputs["Settings"], "env", []byte(e), true)
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

// ParseEnvLine returns valid key, value pair, or an error if an invalid line
func ParseEnvLine(line string) (string, string, error) {
	// Deal with empty lines
	if regexp.MustCompile(`^\s*$`).MatchString(line) {
		return "", "", nil
	}

	// Deal with simple comment lines
	if regexp.MustCompile(`^\s*#.*$`).MatchString(line) {
		return "", "", nil
	}

	// check for invalid lines
	re := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(.*)\s*$`)
	if !re.MatchString(line) {
		return "", "", fmt.Errorf("Invalid env format, expecting key=value: `%s`", line)
	}

	ms := re.FindStringSubmatch(line)
	key := ms[1]

	value := strings.TrimSpace(ms[2])
	value = strings.Trim(value, "'") // heroku env -s adds leading and trailing single quotes so let's strip.
	value = strings.TrimSpace(value)

	return key, value, nil
}
