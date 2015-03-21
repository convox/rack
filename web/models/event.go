package models

import (
  "time"

  "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
  "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
)

type Event struct {
  Id   string
  Name string

  Reason string
  Status string
  Type   string

  Time time.Time
}

type Events []Event

func ListEvents(app string) (Events, error) {
  events := Events{}

  next := ""

  for {
    req := &cloudformation.DescribeStackEventsInput{StackName: aws.String(app)}

    if next != "" {
      req.NextToken = aws.String(next)
    }

    res, err := CloudFormation.DescribeStackEvents(req)

    if err != nil {
      return nil, err
    }

    for _, event := range res.StackEvents {
      events = append(events, Event{
        Id:     *event.EventID,
        Name:   coalesce(event.LogicalResourceID, ""),
        Status: coalesce(event.ResourceStatus, ""),
        Type:   coalesce(event.ResourceType, ""),
        Reason: coalesce(event.ResourceStatusReason, ""),
        Time:   event.Timestamp,
      })
    }

    if res.NextToken == nil {
      break
    }

    next = *res.NextToken
  }

  return events, nil
}