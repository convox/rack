package models

import (
	"sort"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
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

func GroupEvents(events Events) (Transactions, error) {
	name_events := make(map[string][]Event)

	for _, event := range events {
		name_events[event.Name] = append(name_events[event.Name], event)
	}

	transactions := make(Transactions, 0)

	for name, events := range name_events {
		first := events[len(events)-1]
		last := events[0]

		transactions = append(transactions, Transaction{
			Name:   name,
			Type:   first.Type,
			Start:  first.Time,
			End:    last.Time,
			Status: last.Status,
		})
	}

	sort.Sort(transactions)

	return transactions, nil
}

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
