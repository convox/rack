package models

import (
	"fmt"
	"io/ioutil"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
)

type Cluster struct {
	Name string

	AvailabilityZones string
	Count             string
	Key               string
	Size              string
	Subnets           string
	Status            string
	Vpc               string
}

type Clusters []Cluster

func ListClusters() (Clusters, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{})

	if err != nil {
		return nil, err
	}

	clusters := make(Clusters, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "cluster" {
			clusters = append(clusters, *clusterFromStack(stack))
		}
	}

	return clusters, nil
}

func GetCluster(name string) (*Cluster, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(name)})

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	cluster := clusterFromStack(res.Stacks[0])

	return cluster, nil
}

func (c *Cluster) Create() error {
	formation, err := c.Formation()

	if err != nil {
		return err
	}

	params := map[string]string{
		"AvailabilityZones": c.AvailabilityZones,
		"Count":             c.Count,
		"Key":               c.Key,
		"Size":              c.Size,
	}

	tags := map[string]string{
		"System": "convox",
		"Type":   "cluster",
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(c.Name),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err = CloudFormation().CreateStack(req)

	return err
}

func (c *Cluster) Delete() error {
	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(c.Name)})

	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Apps() (Apps, error) {
	return ListAppsByCluster(c.Name)
}

func (c *Cluster) Created() bool {
	return c.Status != "creating"
}

func (c *Cluster) Formation() (string, error) {
	data, err := ioutil.ReadFile("data/cluster.json")

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func clusterFromStack(stack *cloudformation.Stack) *Cluster {
	params := stackParameters(stack)
	outputs := stackOutputs(stack)

	return &Cluster{
		Name:              cs(stack.StackName, "<unknown>"),
		Status:            humanStatus(*stack.StackStatus),
		AvailabilityZones: params["AvailabilityZones"],
		Count:             params["Count"],
		Key:               params["Key"],
		Subnets:           outputs["Subnets"],
		Size:              params["Size"],
		Vpc:               outputs["Vpc"],
	}
}
