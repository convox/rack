package aws

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/api/structs"
)

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}

func (p *AWSProvider) ReleaseGet(app, id string) (*structs.Release, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(app)),
	}

	res, err := p.dynamodb().GetItem(req)

	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such release: %s", id)
	}

	release := releaseFromItem(res.Item)

	return release, nil
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) *structs.Release {
	created, _ := time.Parse(SortableTime, coalesce(item["created"], ""))

	release := &structs.Release{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Build:    coalesce(item["build"], ""),
		Env:      coalesce(item["env"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}

	return release
}
