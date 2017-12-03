package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) EnvironmentGet(app string) (structs.Environment, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	if a.Status == "creating" {
		return nil, fmt.Errorf("app is still being created: %s", app)
	}

	settings, err := p.appResource(app, "Settings")
	if err != nil {
		return nil, err
	}

	data, err := p.s3Get(settings, "env")
	if err != nil {
		// if we get a 404 from aws just return an empty environment
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			return structs.Environment{}, nil
		}

		return nil, err
	}

	key, err := p.rackResource("EncryptionKey")
	if err != nil {
		return nil, err
	}

	if key != "" {
		if d, err := crypt.New().Decrypt(key, data); err == nil {
			data = d
		}
	}

	env := structs.Environment{}

	if err := env.Load(data); err != nil {
		return nil, err
	}

	return env, nil
}

func (p *AWSProvider) EnvironmentPut(app string, env structs.Environment) (string, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return "", err
	}

	// only allow running and updating status through
	switch a.Status {
	case "running", "updating":
	default:
		return "", fmt.Errorf("unable to set environment with current app status: %s, status must be running or updating", a.Status)
	}

	release := structs.NewRelease(app)

	if a.Release != "" {
		r, err := p.ReleaseGet(a.Name, a.Release)
		if err != nil {
			return "", err
		}
		release = r
	}

	release.Id = generateId("R", 10)
	release.Env = env.String()

	if err := p.ReleaseSave(release); err != nil {
		return "", err
	}

	e := []byte(env.String())

	key, err := p.rackResource("EncryptionKey")
	if err != nil {
		return "", err
	}

	if key != "" {
		e, err = crypt.New().Encrypt(key, e)

		if err != nil {
			return "", err
		}
	}

	settings, err := p.appResource(app, "Settings")
	if err != nil {
		return "", err
	}

	err = p.s3Put(settings, "env", []byte(e), false)
	if err != nil {
		return "", err
	}

	return release.Id, nil
}
