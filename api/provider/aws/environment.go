package aws

import (
	"fmt"
	"os"

	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) EnvironmentGet(app string) (structs.Environment, error) {
	a, err := p.AppGet(app)

	if err != nil {
		return nil, err
	}

	data, err := p.s3Get(a.Outputs["Settings"], "env")

	// return blank environment if no environment file on s3 yet
	if awsError(err) == "NoSuchKey" {
		return structs.Environment{}, nil
	}

	if err != nil {
		return nil, err
	}

	if a.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		if d, err := cr.Decrypt(a.Parameters["Key"], data); err == nil {
			data = d
		}
	}

	return structs.LoadEnvironment(data), nil
}

func (p *AWSProvider) EnvironmentSet(app string, env structs.Environment) (string, error) {
	a, err := p.AppGet(app)

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

	release, err := p.ReleaseFork(a.Name)

	if err != nil {
		return "", err
	}

	release.Env = env.Raw()

	err = p.ReleaseSave(release)

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

	err = p.s3Put(a.Outputs["Settings"], "env", []byte(e), true)

	if err != nil {
		return "", err
	}

	return release.Id, nil
}
