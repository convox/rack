package models

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aryann/difflib"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/convox/env/crypt"
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

func PutEnvironment(app string, env Environment) error {
	a, err := GetApp(app)

	if err != nil {
		return err
	}

	release, err := a.ForkRelease()

	if err != nil {
		return err
	}

	release.Env = env.Raw()

	err = release.Save()

	if err != nil {
		return err
	}

	// eold := strings.Split(release.Env, "\n")
	// enew := strings.Split(env.Raw(), "\n")
	// diff := difflib.Diff(eold, enew)

	// metadata, err := diffMetadata(diff)

	// if err != nil {
	//   return err
	// }

	// change := &Change{
	//   App:      app,
	//   Created:  time.Now(),
	//   Metadata: metadata,
	//   TargetId: release.Id,
	//   Type:     "RELEASE",
	//   Status:   "complete",
	//   User:     "convox",
	// }

	// err = change.Save()

	// if err != nil {
	//   fmt.Fprintf(os.Stderr, "error: %s\n", err)
	// }

	e := []byte(env.Raw())

	if a.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		e, err = cr.Encrypt(a.Parameters["Key"], e)

		if err != nil {
			return err
		}
	}

	return S3Put(a.Outputs["Settings"], "env", []byte(e), true)
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

func diffMetadata(diff []difflib.DiffRecord) (string, error) {
	changes := map[string]string{}

	for _, d := range diff {
		parts := strings.SplitN(d.Payload, "=", 2)

		if len(parts) == 2 {
			switch d.Delta {
			case difflib.RightOnly:
				switch changes[parts[0]] {
				case "deleted", "changed":
					changes[parts[0]] = "changed"
				default:
					changes[parts[0]] = "added"
				}
			case difflib.LeftOnly:
				switch changes[parts[0]] {
				case "added", "changed":
					changes[parts[0]] = "changed"
				default:
					changes[parts[0]] = "deleted"
				}
			}
		}
	}

	names := []string{}

	for name, _ := range changes {
		names = append(names, name)
	}

	sort.Strings(names)

	meta := ChangeMetadata{}

	meta.Transactions = make([]Transaction, len(names))

	for i, name := range names {
		meta.Transactions[i] = Transaction{
			Name:   name,
			Type:   "Env::Diff",
			Status: changes[name],
		}
	}

	data, err := json.Marshal(meta)

	if err != nil {
		return "", err
	}

	return string(data), nil
}
