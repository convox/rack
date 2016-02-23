package models

import (
	"encoding/json"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

// Use the Rack Settings bucket and EncryptionKey KMS key to store and retrieve
// sensitive credentials, just like app env
func GetRackSettings() (structs.Environment, error) {
	a, err := provider.AppGet(os.Getenv("RACK"))

	if err != nil {
		return nil, err
	}

	resources, err := ListResources(a.Name)

	if err != nil {
		return nil, err
	}

	key := resources["EncryptionKey"].Id
	settings := resources["Settings"].Id

	data, err := s3Get(settings, "env")

	if err != nil {
		// if we get a 404 from aws just return an empty environment
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			return structs.Environment{}, nil
		}

		return nil, err
	}

	if key != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		if d, err := cr.Decrypt(key, data); err == nil {
			data = d
		}
	}

	var env structs.Environment
	err = json.Unmarshal(data, &env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func PutRackSettings(env structs.Environment) error {
	a, err := provider.AppGet(os.Getenv("RACK"))

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

	if err != nil {
		return err
	}

	return nil
}
