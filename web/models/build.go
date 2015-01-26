package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Build struct {
	Id string

	Logs    string
	Release string
	Status  string

	Created time.Time
	Ended   time.Time
}

type Builds []Build

func ListBuilds(app string) (Builds, error) {
	table := buildsTable(app)

	q := dynamodb.NewQuery(table)
	q.AddIndex("app.created")
	q.AddKeyConditions([]dynamodb.AttributeComparison{
		*dynamodb.NewEqualStringAttributeComparison("app", app),
	})
	q.AddScanIndexForward(false)
	q.AddLimit(5)

	rows, _, err := table.QueryTable(q)

	if err != nil {
		return nil, err
	}

	builds := make(Builds, len(rows))

	for i, row := range rows {
		builds[i] = *buildFromRow(row)
	}

	return builds, nil
}

func buildFromRow(row map[string]*dynamodb.Attribute) *Build {
	created, _ := time.Parse(SortableTime, coalesce(row["created"], ""))
	ended, _ := time.Parse(SortableTime, coalesce(row["ended"], ""))

	return &Build{
		Id:      coalesce(row["id"], ""),
		Logs:    coalesce(row["logs"], ""),
		Release: coalesce(row["release"], ""),
		Status:  coalesce(row["status"], ""),
		Created: created,
		Ended:   ended,
	}
}

func buildsTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-builds", app), pk)
	return table
}
