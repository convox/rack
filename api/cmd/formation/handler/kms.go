package handler

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms"
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
		return "invalid", nil, err
	}

	return *res.KeyMetadata.Arn, nil, nil
}

func KMSKeyUpdate(req Request) (string, map[string]string, error) {
	return req.PhysicalResourceId, nil, nil
}

func KMSKeyDelete(req Request) (string, map[string]string, error) {
	_, err := KMS(req).DisableKey(&kms.DisableKeyInput{
		KeyId: aws.String(req.PhysicalResourceId),
	})

	// go ahead and mark the delete good if the key is not found
	if ae, ok := err.(awserr.Error); ok {
		if ae.Code() == "NotFoundException" {
			return req.PhysicalResourceId, nil, nil
		}
	}

	if err != nil {
		fmt.Printf("error: %s\n", err)
		return req.PhysicalResourceId, nil, err
	}

	_, err = KMS(req).ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
		KeyId:               aws.String(req.PhysicalResourceId),
		PendingWindowInDays: aws.Int64(7),
	})

	return req.PhysicalResourceId, nil, err
}
