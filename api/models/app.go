package models

import (
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

// var CustomTopic = os.Getenv("CUSTOM_TOPIC")

// var StatusCodePrefix = client.StatusCodePrefix

// Shortcut for updating current parameters
// If template changed, more care about new or removed parameters must be taken (see Release.Promote or System.Save)
func AppUpdateParams(app *structs.App, changes map[string]string) error {
	req := &cloudformation.UpdateStackInput{
		StackName:           aws.String(app.Name),
		Capabilities:        []*string{aws.String("CAPABILITY_IAM")},
		UsePreviousTemplate: aws.Bool(true),
	}

	params := app.Parameters

	for key, val := range changes {
		params[key] = val
	}

	for key, val := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(val),
		})
	}

	_, err := CloudFormation().UpdateStack(req)

	return err
}

func appForkRelease(app *structs.App) (*structs.Release, error) {
	release, err := appLatestRelease(app)

	if err != nil {
		return nil, err
	}

	if release == nil {
		r := NewRelease(app.Name)
		release = &r
	}

	release.Id = generateId("R", 10)
	release.Created = time.Time{}

	return release, nil
}

func appLatestRelease(app *structs.App) (*structs.Release, error) {
	releases, err := provider.ReleaseList(app.Name)

	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return &releases[0], nil
}
