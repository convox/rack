package provider

import "fmt"

type Cluster struct {
	Name   string
	Id     string
	Status string

	Apps []App
}

func ClusterList() ([]Cluster, error) {
	res, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return nil, err
	}

	clusters := make([]Cluster, 0)

	for _, stack := range res.Stacks {
		if flattenTags(stack.Tags)["type"] == "cluster" {
			clusters = append(clusters, Cluster{Name: stack.StackName, Status: humanStatus(stack.StackStatus)})
		}
	}

	return clusters, nil
}

func ClusterCreate(name string) error {
	cluster := &Cluster{Name: name}

	formation, err := buildTemplate("cluster", cluster)

	if err != nil {
		return err
	}

	tags := map[string]string{
		"type":    "cluster",
		"cluster": name,
	}

	err = createStackFromTemplate(formation, name, tags)

	if err != nil {
		return fmt.Errorf("could not create stack %s: %s", name, err)
	}

	return nil
}

func ClusterDelete(name string) error {
	_, err := CloudFormation.DeleteStack(name)

	if err != nil {
		return fmt.Errorf("could not delete stack %s: %s", name, err)
	}

	return nil
}
