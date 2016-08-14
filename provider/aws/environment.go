package aws

import (
	"fmt"
	"os"

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

	data, err := p.s3Get(a.Outputs["Settings"], "env")
	if err != nil {
		// if we get a 404 from aws just return an empty environment
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			return structs.Environment{}, nil
		}

		return nil, err
	}

	if a.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		if d, err := cr.Decrypt(a.Parameters["Key"], data); err == nil {
			data = d
		}
	}

	env := structs.Environment{}
	env.LoadEnvironment(data)

	return env, nil
}
