package models

import (
	"sort"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
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

type ServiceEvent struct {
	Id        string
	Message   string
	CreatedAt time.Time
}

type ServiceEvents []ServiceEvent

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

		res, err := CloudFormation().DescribeStackEvents(req)

		if err != nil {
			return nil, err
		}

		for _, event := range res.StackEvents {
			events = append(events, Event{
				Id:     cs(event.EventID, ""),
				Name:   cs(event.LogicalResourceID, ""),
				Status: cs(event.ResourceStatus, ""),
				Type:   cs(event.ResourceType, ""),
				Reason: cs(event.ResourceStatusReason, ""),
				Time:   ct(event.Timestamp),
			})
		}

		if res.NextToken == nil {
			break
		}

		next = *res.NextToken
	}

	return events, nil
}

func ListECSEvents(app string) (ServiceEvents, error) {
	events := ServiceEvents{}

	req := &ecs.DescribeServicesInput{
		Cluster:  aws.String("testing"),
		Services: []*string{aws.String(app)},
	}

	res, err := ECS().DescribeServices(req)

	if err != nil {
		return nil, err
	}

	for _, event := range res.Services[0].Events {
		events = append(events, ServiceEvent{
			Id:        cs(event.ID, ""),
			Message:   cs(event.Message, ""),
			CreatedAt: ct(event.CreatedAt),
		})
	}

	return events, nil
}

func (slice ServiceEvents) Len() int {
	return len(slice)
}

func (slice ServiceEvents) Less(i, j int) bool {
	return slice[i].CreatedAt.Before(slice[j].CreatedAt)
}

func (slice ServiceEvents) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
