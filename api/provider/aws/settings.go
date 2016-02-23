package aws

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) SettingsGet(app string) (structs.Settings, error) {
	resources, err := p.stackResources(os.Getenv("RACK"))

	if err != nil {
		return nil, err
	}

	if resources["Settings"].Id == "" {
		return nil, fmt.Errorf("no settings bucket")
	}

	bucket := resources["Settings"].Id

	data, err := p.s3Get(bucket, "env")

	// if we get a 404 from aws just return empty settings
	if awsError(err) == "NoSuchKey" {
		return structs.Settings{}, nil
	}

	if err != nil {
		return nil, err
	}

	if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		if d, err := cr.Decrypt(key, data); err == nil {
			data = d
		}
	}

	settings, err := structs.LoadSettings(data)

	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (p *AWSProvider) SettingsSet(app string, settings structs.Settings) error {
	resources, err := p.stackResources(app)

	if err != nil {
		return err
	}

	if resources["Settings"].Id == "" {
		return fmt.Errorf("no settings bucket")
	}

	bucket := resources["Settings"].Id

	data, err := json.Marshal(settings)

	if err != nil {
		return err
	}

	if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		data, err = cr.Encrypt(key, data)

		if err != nil {
			return err
		}
	}

	err = p.s3Put(bucket, "env", data, true)

	if err != nil {
		return err
	}

	return nil
}
