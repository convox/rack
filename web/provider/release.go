package provider

import (
	"fmt"
	"strings"
)

func ReleaseDeploy(cluster, app, release string) error {
	outputs, err := stackOutputs(cluster)

	if err != nil {
		return err
	}

	vpc := outputs["Vpc"]
	rt := outputs["RouteTable"]

	base, err := nextAvailableSubnet(vpc)

	if err != nil {
		return err
	}

	params := &AppParams{
		Name:    upperName(app),
		Cluster: upperName(cluster),
		Cidr:    base,
		Vpc:     vpc,
	}

	azs, err := availabilityZones()

	if err != nil {
		return err
	}

	subnets, err := divideSubnet(base, len(azs))

	if err != nil {
		return err
	}

	params.Subnets = make([]AppParamsSubnet, len(azs))

	for i, az := range azs {
		params.Subnets[i] = AppParamsSubnet{
			Name:             fmt.Sprintf("Subnet%d", i),
			AvailabilityZone: az,
			Cidr:             subnets[i],
			RouteTable:       rt,
			Vpc:              vpc,
		}
	}

	uparams := UserdataParams{
		Process:   "web",
		Env:       map[string]string{"FOO": "bar"},
		Resources: []UserdataParamsResource{},
		Ports:     []int{5000},
	}

	userdata, err := parseTemplate("userdata", uparams)

	if err != nil {
		return err
	}

	params.Processes = []AppParamsProcess{
		{
			Name:              "Web",
			Process:           "web",
			Count:             2,
			Vpc:               vpc,
			App:               app,
			Ami:               "ami-acb1cfc4",
			Cluster:           cluster,
			AvailabilityZones: azs,
			UserData:          userdata,
		},
	}

	formation, err := parseTemplate("app", params)

	lines := strings.Split(formation, "\n")

	for i, line := range lines {
		fmt.Printf("%d: %s\n", i, line)
	}

	if err != nil {
		return err
	}

	tags := map[string]string{
		"type":    "app",
		"cluster": cluster,
		"app":     app,
		"subnet":  base,
	}

	return createStackFromTemplate(formation, fmt.Sprintf("%s-%s", cluster, app), tags)
}
