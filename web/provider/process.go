package provider

import (
	"fmt"
	"strconv"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Process struct {
	Name  string
	Count int
}

func ProcessCreate(cluster, app, process string) error {
	attributes := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", process),
		*dynamodb.NewStringAttribute("count", "1"),
	}

	_, err := processesTable(cluster, app).PutItem(process, "", attributes)

	return err
}

func ProcessList(cluster, app string) ([]Process, error) {
	res, err := processesTable(cluster, app).Scan(nil)

	if err != nil {
		return nil, err
	}

	processes := make([]Process, len(res))

	for i, ps := range res {
		count, err := strconv.Atoi(coalesce(ps["count"], "0"))

		if err != nil {
			return nil, err
		}

		processes[i] = Process{
			Name:  coalesce(ps["name"], ""),
			Count: count,
		}
	}

	return processes, nil
}

func processesTable(cluster, app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-%s-processes", cluster, app), pk)
	return table
}
