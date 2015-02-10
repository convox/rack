package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

type Release struct {
	Id string

	Ami      string
	Manifest string
	Status   string

	App string

	Created time.Time
}

type Releases []Release

func ListReleases(app string) (Releases, error) {
	table := releasesTable(app)

	q := dynamodb.NewQuery(table)
	q.AddIndex("app.created")
	q.AddKeyConditions([]dynamodb.AttributeComparison{
		*dynamodb.NewEqualStringAttributeComparison("app", app),
	})
	q.AddScanIndexForward(false)
	q.AddLimit(10)

	rows, _, err := table.QueryTable(q)

	if err != nil {
		return nil, err
	}

	releases := make(Releases, len(rows))

	for i, row := range rows {
		releases[i] = *releaseFromRow(row)
	}

	return releases, nil
}

func GetRelease(app, id string) (*Release, error) {
	row, err := releasesTable(app).GetItem(&dynamodb.Key{id, ""})

	if err != nil {
		return nil, err
	}

	return releaseFromRow(row), nil
}

func (r *Release) Formation() (string, error) {
	app, err := GetApp(r.App)

	if err != nil {
		return "", err
	}

	app.Release = r.Id

	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return "", err
	}

	err = manifest.Apply(app)

	if err != nil {
		return "", err
	}

	formation, err := app.Formation()

	if err != nil {
		return "", err
	}

	return formation, nil
}

func (r *Release) Promote() error {
	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	formation, err := r.Formation()

	if err != nil {
		return err
	}

	sp := &cloudformation.UpdateStackParams{
		StackName:    fmt.Sprintf("convox-%s", r.App),
		TemplateBody: formation,
		Capabilities: []string{"CAPABILITY_IAM"},
		Parameters: []cloudformation.Parameter{
			cloudformation.Parameter{ParameterKey: "Release", ParameterValue: r.Id},
			cloudformation.Parameter{ParameterKey: "Repository", ParameterValue: app.Repository},
		},
	}

	_, err = CloudFormation.UpdateStack(sp)

	return err
}

func releasesTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-releases", app), pk)
	return table
}

func releaseFromRow(row map[string]*dynamodb.Attribute) *Release {
	created, _ := time.Parse(SortableTime, coalesce(row["created"], ""))

	return &Release{
		Id:       coalesce(row["id"], ""),
		Ami:      coalesce(row["ami"], ""),
		Manifest: coalesce(row["manifest"], ""),
		Status:   coalesce(row["status"], ""),
		App:      coalesce(row["app"], ""),
		Created:  created,
	}
}
