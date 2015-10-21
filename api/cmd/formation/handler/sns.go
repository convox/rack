package handler

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)

func HandleSNSSubcription(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING SNSSUBSCRIPTION")
		fmt.Printf("req %+v\n", req)
		return SNSSubscriptionCreate(req)
	case "Update":
		fmt.Println("UPDATING SNSSUBSCRIPTION")
		fmt.Printf("req %+v\n", req)
		return SNSSubscriptionUpdate(req)
	case "Delete":
		fmt.Println("DELETING SNSSUBSCRIPTION")
		fmt.Printf("req %+v\n", req)
		return SNSSubscriptionDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func SNSSubscriptionCreate(req Request) (string, map[string]string, error) {
	endpoint := req.ResourceProperties["Url"].(string)

	res, err := SNS(req).Subscribe(&sns.SubscribeInput{
		TopicArn: aws.String(endpoint),
		Endpoint: aws.String(req.ResourceProperties["Endpoint"].(string)),
		Protocol: aws.String(req.ResourceProperties["Protocol"].(string)),
	})

	if err != nil {
		return "", nil, err
	}

	outputs := make(map[string]string)
	outputs["Endpoint"] = endpoint

	return res.SubscriptionArn, outputs, nil
}

func SNSSubscriptionUpdate(req Request) (string, map[string]string, error) {
	_, _, err := SNSSubscriptionDelete(req)
	if err != nil {
		return "", nil, err
	}

	return SNSSubscriptionCreate(req)
}

func SNSSubscriptionDelete(req Request) (string, map[string]string, error) {
	res, err := SNS(req).Unsubscribe(&sns.UnsubscribeInput{
		SubscriptionArn: aws.String(req.PhysicalResourceId),
	})

	return req.PhysicalResourceId, nil, err
}
