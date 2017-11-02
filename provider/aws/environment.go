package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
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
	env.LoadEnvironment(data)

	return env, nil
}
