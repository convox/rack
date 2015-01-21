package provider

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Release struct {
	Ami       string
	CreatedAt time.Time
}

type UserdataParams struct {
	Process   string
	Env       map[string]string
	Resources []UserdataParamsResource
	Ports     []int
}

type UserdataParamsResource struct {
}

func ReleaseList(cluster, app string) ([]Release, error) {
	res, err := releasesTable(cluster, app).Scan(nil)

	if err != nil {
		return nil, err
	}

	releases := []Release{}

	for _, r := range res {
		releases = append(releases, Release{
			Ami:       r["ami"].Value,
			CreatedAt: time.Now(),
		})
	}

	return releases, nil
}

func ReleaseCreate(cluster, app, ami string, options map[string]string) error {
	attributes := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("ami", ami),
		*dynamodb.NewStringAttribute("created-at", "now"),
	}

	for k, v := range options {
		attributes = append(attributes, *dynamodb.NewStringAttribute(k, v))
	}

	_, err := releasesTable(cluster, app).PutItem(ami, "", attributes)

	return err
}

func ReleaseDeploy(cluster, app, ami string) error {
	params, err := appParams(cluster, app)

	if err != nil {
		return err
	}

	uparams := UserdataParams{
		Process:   "web",
		Env:       map[string]string{"FOO": "bar"},
		Resources: []UserdataParamsResource{},
		Ports:     []int{5000},
	}

	userdata, err := buildTemplate("userdata", uparams)

	if err != nil {
		return err
	}

	params.Processes = []AppParamsProcess{
		{
			Ami:               ami,
			App:               app,
			AvailabilityZones: params.AvailabilityZones,
			Cluster:           cluster,
			Count:             2,
			Name:              "web",
			UserData:          userdata,
			Vpc:               params.Vpc,
		},
	}

	template, err := buildTemplate("app", params)

	if err != nil {
		return err
	}

	printLines(template)

	// update stack

	return nil
}

func releasesTable(cluster, app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("ami", ""), dynamodb.NewStringAttribute("created-at", "")}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-%s-releases", cluster, app), pk)
	return table
}
