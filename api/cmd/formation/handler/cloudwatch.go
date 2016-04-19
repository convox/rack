package handler

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
)

func HandleCloudWatchEventsRule(req Request) (string, map[string]string, error) {
	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING RULE")
		fmt.Printf("req %+v\n", req)
		return CloudWatchEventsRuleCreate(req)
	case "Update":
		fmt.Println("UPDATING RULE")
		fmt.Printf("req %+v\n", req)
		return CloudWatchEventsRuleUpdate(req)
	case "Delete":
		fmt.Println("DELETING RULE")
		fmt.Printf("req %+v\n", req)
		return CloudWatchEventsRuleDelete(req)
	}

	return "invalid", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func CloudWatchEventsRuleCreate(req Request) (string, map[string]string, error) {
	res, err := CloudWatchEvents(req).PutRule(&cloudwatchevents.PutRuleInput{
		Name:               aws.String(req.ResourceProperties["Name"].(string)),
		ScheduleExpression: aws.String(req.ResourceProperties["ScheduleExpression"].(string)),
	})

	if err != nil {
		return "invalid", nil, err
	}

	return *res.RuleArn, nil, nil
}

func CloudWatchEventsRuleUpdate(req Request) (string, map[string]string, error) {
	return req.PhysicalResourceId, nil, nil
}

func CloudWatchEventsRuleDelete(req Request) (string, map[string]string, error) {
	return req.PhysicalResourceId, nil, nil
}
