package handler

import (
	"fmt"
	"os"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/kms"
)

func HandleKMSKey(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING KEY")
		fmt.Printf("req %+v\n", req)
		return KMSKeyCreate(req)
	case "Update":
		fmt.Println("UPDATING KEY")
		fmt.Printf("req %+v\n", req)
		return KMSKeyUpdate(req)
	case "Delete":
		fmt.Println("DELETING KEY")
		fmt.Printf("req %+v\n", req)
		return KMSKeyDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func KMSKeyCreate(req Request) (string, map[string]string, error) {
	res, err := KMS(req).CreateKey(&kms.CreateKeyInput{
		Description: aws.String(req.ResourceProperties["Description"].(string)),
		KeyUsage:    aws.String(req.ResourceProperties["KeyUsage"].(string)),
	})

	if err != nil {
		return "", nil, err
	}

	return *res.KeyMetadata.ARN, nil, nil
}

func KMSKeyUpdate(req Request) (string, map[string]string, error) {
	return req.PhysicalResourceId, nil, fmt.Errorf("could not update")
}

func KMSKeyDelete(req Request) (string, map[string]string, error) {
	_, err := KMS(req).DisableKey(&kms.DisableKeyInput{
		KeyID: aws.String(req.PhysicalResourceId),
	})

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return req.PhysicalResourceId, nil, nil
	}

	return req.PhysicalResourceId, nil, nil
}
