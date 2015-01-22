package provider

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Build struct {
	Id        string
	Status    string
	Release   string
	CreatedAt time.Time
	EndedAt   time.Time
	Logs      string
}

func BuildsList(cluster string, app string) ([]Build, error) {
	table := buildsTable(cluster, app)

	q := dynamodb.NewQuery(table)
	q.AddIndex("app.created")
	q.AddKeyConditions([]dynamodb.AttributeComparison{
		*dynamodb.NewEqualStringAttributeComparison("app", app),
	})
	q.AddScanIndexForward(false)
	q.AddLimit(5)

	dbuilds, _, err := table.QueryTable(q)

	if err != nil {
		return nil, err
	}

	builds := make([]Build, len(dbuilds))

	for i, db := range dbuilds {
		created, err := time.Parse(SortableTime, db["created"].Value)

		if err != nil {
			return nil, err
		}

		var ended time.Time

		if db["ended"] != nil {
			ended, err = time.Parse(SortableTime, db["ended"].Value)

			if err != nil {
				return nil, err
			}
		}

		builds[i] = Build{
			Id:        coalesce(db["id"], ""),
			Status:    coalesce(db["status"], ""),
			Release:   coalesce(db["release"], ""),
			CreatedAt: created,
			EndedAt:   ended,
		}
	}

	return builds, nil
}

func buildsTable(cluster, app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), dynamodb.NewStringAttribute("created-at", "")}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-%s-builds", cluster, app), pk)
	return table
}
