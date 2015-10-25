package handler

import (
	"fmt"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
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
	endpoint := req.ResourceProperties["Endpoint"].(string)
	topicArn := req.ResourceProperties["TopicArn"].(string)

	input := sns.SubscribeInput{
		Endpoint: aws.String(endpoint),
		Protocol: aws.String(req.ResourceProperties["Protocol"].(string)),
		TopicArn: aws.String(topicArn),
	}

	_, err := SNS(req).Subscribe(&input)

	if err != nil {
		return "failed", nil, err
	}

	outputs := make(map[string]string)
	outputs["Endpoint"] = endpoint

	return endpoint, outputs, nil
}

func SNSSubscriptionUpdate(req Request) (string, map[string]string, error) {
	return SNSSubscriptionCreate(req)
}

func SNSSubscriptionDelete(req Request) (string, map[string]string, error) {
	if req.PhysicalResourceId == "failed" {
		return req.PhysicalResourceId, nil, nil
	}

	topicArn := req.ResourceProperties["TopicArn"].(string)
	params := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(topicArn),
	}

	resp, err := SNS(req).ListSubscriptionsByTopic(params)

	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	for _, s := range resp.Subscriptions {
		if *s.Endpoint == req.PhysicalResourceId {
			_, err := SNS(req).Unsubscribe(&sns.UnsubscribeInput{
				SubscriptionArn: aws.String(*s.SubscriptionArn),
			})

			if err != nil {
				fmt.Printf("error: %s\n", err)
				return req.PhysicalResourceId, nil, nil
			}

			return *s.Endpoint, nil, nil
		}
	}

	return req.PhysicalResourceId, nil, nil
}
